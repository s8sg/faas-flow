package main

import (
	"bytes"
	"fmt"
	"handler/function"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	faasflow "github.com/s8sg/faas-flow"

	sdk "github.com/s8sg/faas-flow/sdk"
	executor "github.com/s8sg/faas-flow/sdk/executor"
	exporter "github.com/s8sg/faas-flow/sdk/exporter"
)

const (
	// A signature of SHA265 equivalent of "github.com/s8sg/faas-flow"
	defaultHmacKey          = "71F1D3011F8E6160813B4997BA29856744375A7F26D427D491E1CCABD4627E7C"
	counterUpdateRetryCount = 10
)

var (
	re = regexp.MustCompile(`(?m)^[^:.]+\s*`)
)

// implements faasflow.EventHandler
type openFaasEventHandler struct {
	currentNodeId string        // used to inject current node id in tracer
	tracer        *traceHandler // handle traces with open-tracing
	flowName      string
	header        http.Header
}

// implements faasflow.Logger
type openFaasLogger struct{}

// implements faasflow.Executor + RequestHandler
type openFaasExecutor struct {
	gateway      string
	asyncUrl     string // the async URL of the flow
	flowName     string // the name of the function
	reqId        string // the request id
	partialState []byte
	rawRequest   *executor.RawRequest

	openFaasEventHandler
	openFaasLogger
}

// Logger
func (logger *openFaasLogger) Configure(flowName string, requestId string) {}
func (logger *openFaasLogger) Init() error {
	return nil
}
func (logger *openFaasLogger) Log(str string) {
	fmt.Print(str)
}

// EventHandler

func (eh *openFaasEventHandler) Configure(flowName string, requestId string) {
	eh.flowName = flowName
}

func (eh *openFaasEventHandler) Init() error {
	var err error
	// initialize trace server if tracing enabled
	eh.tracer, err = initRequestTracer(eh.flowName)
	if err != nil {
		return fmt.Errorf("failed to init request tracer, error %v", err)
	}
	return nil
}

func (eh *openFaasEventHandler) ReportRequestStart(requestId string) {
	eh.tracer.startReqSpan(requestId)
}

func (eh *openFaasEventHandler) ReportRequestFailure(requestId string, err error) {
	// TODO: add log
	eh.tracer.stopReqSpan()
}

func (eh *openFaasEventHandler) ReportExecutionForward(currentNodeId string, requestId string) {
	eh.currentNodeId = currentNodeId
}

func (eh *openFaasEventHandler) ReportExecutionContinuation(requestId string) {
	eh.tracer.continueReqSpan(requestId, eh.header)
}

func (eh *openFaasEventHandler) ReportRequestEnd(requestId string) {
	eh.tracer.stopReqSpan()
}

func (eh *openFaasEventHandler) ReportNodeStart(nodeId string, requestId string) {
	eh.tracer.startNodeSpan(nodeId, requestId)
}

func (eh *openFaasEventHandler) ReportNodeEnd(nodeId string, requestId string) {
	eh.tracer.stopNodeSpan(nodeId)
}

func (eh *openFaasEventHandler) ReportNodeFailure(nodeId string, requestId string, err error) {
	// TODO: add log
	eh.tracer.stopNodeSpan(nodeId)
}

func (eh *openFaasEventHandler) ReportOperationStart(operationId string, nodeId string, requestId string) {
	// TODO: add feature
}

func (eh *openFaasEventHandler) ReportOperationEnd(operationId string, nodeId string, requestId string) {
	// TODO: add feature
}

func (eh *openFaasEventHandler) ReportOperationFailure(operationId string, nodeId string, requestId string, err error) {
	// TODO: add feature
}

func (eh *openFaasEventHandler) Flush() {
	eh.tracer.flushTracer()
}

// ExecutionRuntime

func (of *openFaasExecutor) HandleNextNode(partial *executor.PartialState) error {

	state, err := partial.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode partial state, error %v", err)
	}

	// build url for calling the flow in async
	httpreq, _ := http.NewRequest(http.MethodPost, of.asyncUrl, bytes.NewReader(state))
	httpreq.Header.Add("Accept", "application/json")
	httpreq.Header.Add("Content-Type", "application/json")
	httpreq.Header.Add("X-Faas-Flow-Reqid", of.reqId)
	httpreq.Header.Set("X-Faas-Flow-State", "partial")

	// extend req span for async call
	of.tracer.extendReqSpan(of.reqId, of.openFaasEventHandler.currentNodeId,
		of.asyncUrl, httpreq)

	client := &http.Client{}
	res, resErr := client.Do(httpreq)
	if resErr != nil {
		return resErr
	}

	defer res.Body.Close()
	resdata, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("%d: %s", res.StatusCode, string(resdata))
	}
	return nil
}

func (of *openFaasExecutor) GetExecutionOption(operation sdk.Operation) map[string]interface{} {
	options := make(map[string]interface{})
	options["gateway"] = of.gateway
	options["request-id"] = of.reqId

	return options
}

// Executor

func (of *openFaasExecutor) Configure(requestId string) {
	of.reqId = requestId
}

func (of *openFaasExecutor) GetFlowName() string {
	return of.flowName
}

func (of *openFaasExecutor) GetFlowDefinition(pipeline *sdk.Pipeline, context *sdk.Context) error {
	workflow := faasflow.GetWorkflow(pipeline)
	faasflowContext := (*faasflow.Context)(context)
	err := function.Define(workflow, faasflowContext)
	return err
}

func (of *openFaasExecutor) ReqValidationEnabled() bool {
	status := true
	hmacStatus := os.Getenv("validate_request")
	if strings.ToUpper(hmacStatus) == "FALSE" {
		status = false
	}
	return status
}

func (of *openFaasExecutor) GetValidationKey() (string, error) {
	key, keyErr := readSecret("faasflow-hmac-secret")
	if keyErr != nil {
		key = defaultHmacKey
	}
	return key, nil
}

func (of *openFaasExecutor) ReqAuthEnabled() bool {
	status := false
	verifyStatus := os.Getenv("authenticate_request")
	if strings.ToUpper(verifyStatus) == "TRUE" {
		status = true
	}
	return status
}

func (of *openFaasExecutor) GetReqAuthKey() (string, error) {
	key, keyErr := readSecret("faasflow-hmac-secret")
	return key, keyErr
}

func (of *openFaasExecutor) MonitoringEnabled() bool {
	tracing := os.Getenv("enable_tracing")
	if strings.ToUpper(tracing) == "TRUE" {
		return true
	}
	return false
}

func (of *openFaasExecutor) GetEventHandler() (sdk.EventHandler, error) {
	return &of.openFaasEventHandler, nil
}

func (of *openFaasExecutor) LoggingEnabled() bool {
	return true
}

func (of *openFaasExecutor) GetLogger() (sdk.Logger, error) {
	return &of.openFaasLogger, nil
}

func (of *openFaasExecutor) GetStateStore() (sdk.StateStore, error) {
	return function.DefineStateStore()
}

func (of *openFaasExecutor) GetDataStore() (sdk.DataStore, error) {
	return function.DefineDataStore()
}

// internal

func (of *openFaasExecutor) init(req *HttpRequest) error {

	of.gateway = getGateway()
	of.flowName = getWorkflowNameFromHost(req.Host)
	if of.flowName == "" {
		return fmt.Errorf("failed to parse workflow name from host")
	}
	of.asyncUrl = buildURL("http://"+of.gateway, "async-function", of.flowName)

	var err error
	// initialize trace server if tracing enabled
	of.openFaasEventHandler.tracer, err = initRequestTracer(of.flowName)
	if err != nil {
		return fmt.Errorf("failed to init request tracer, error %v", err)
	}

	return nil
}

// Handle handle requests to flow function
func (of *openFaasExecutor) Handle(req *HttpRequest, response *HttpResponse) error {

	err := of.init(req)
	if err != nil {
		panic(err.Error())
	}

	switch {
	case isDagExportRequest(req):
		flowExporter := exporter.CreateFlowExporter(of)
		resp, err := flowExporter.Export()
		if err != nil {
			return fmt.Errorf("failed to export dag, error %v", err)
		}
		response.Body = resp

	case getStopRequestId(req) != "":
		requestId := getStopRequestId(req)
		flowExecutor := executor.CreateFlowExecutor(of)
		err := flowExecutor.Stop(requestId)
		if err != nil {
			log.Printf(err.Error())
			return fmt.Errorf("Failed to stop request " + requestId + ", check if request is active")
		}
		response.Body = []byte("Successfully stopped request " + requestId)

	case getPauseRequestId(req) != "":
		requestId := getPauseRequestId(req)
		flowExecutor := executor.CreateFlowExecutor(of)
		err := flowExecutor.Pause(requestId)
		if err != nil {
			log.Printf(err.Error())
			return fmt.Errorf("Failed to pause request " + requestId + ", check if request is active")
		}
		response.Body = []byte("Successfully paused request " + requestId)

	case getResumeRequestId(req) != "":
		requestId := getResumeRequestId(req)
		flowExecutor := executor.CreateFlowExecutor(of)
		err := flowExecutor.Resume(requestId)
		if err != nil {
			log.Printf(err.Error())
			return fmt.Errorf("Failed to resume request " + requestId + ", check if request is active")
		}
		response.Body = []byte("Successfully resumed request " + requestId)

	default:
		var stateOption executor.ExecutionStateOption

		requestId := req.Header.Get("X-Faas-Flow-Reqid")
		state := req.Header.Get("X-Faas-Flow-State")
		if state == "" {
			rawRequest := &executor.RawRequest{}
			rawRequest.Data = req.Body
			rawRequest.Query = req.QueryString
			rawRequest.AuthSignature = req.Header.Get("X-Hub-Signature")
			// Check if any request Id is passed
			if requestId != "" {
				rawRequest.RequestId = requestId
			}
			stateOption = executor.NewRequest(rawRequest)
		} else {
			if requestId == "" {
				return fmt.Errorf("request ID not set in partial request")
			}
			of.openFaasEventHandler.header = req.Header
			partialState, err := executor.DecodePartialReq(req.Body)
			if err != nil {
				log.Printf(err.Error())
				return fmt.Errorf("failed to decode partial state")
			}
			stateOption = executor.PartialRequest(partialState)
		}

		// Create a flow executor, OpenFaaSExecutor implements executor
		flowExecutor := executor.CreateFlowExecutor(of)
		resp, err := flowExecutor.Execute(stateOption)
		if err != nil {
			log.Printf(err.Error())
			return fmt.Errorf("failed to execute request")
		}
		response.Body = resp
		response.Header.Set("X-Faas-Flow-Reqid", of.reqId)
	}

	response.StatusCode = http.StatusOK
	return nil
}
