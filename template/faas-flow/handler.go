package main

import (
	"bytes"
	"fmt"
	"handler/function"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	faasflow "github.com/s8sg/faas-flow"

	sdk "github.com/s8sg/faas-flow/sdk"
	executor "github.com/s8sg/faas-flow/sdk/executor"
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
	tracer        *traceHandler // handle traces with opentracing
	flowName      string
}

// implements faasflow.Logger
type openFaasLogger struct{}

// implements faasflow.Executor + RequestHandler
type openFaasExecutor struct {
	gateway  string
	asyncUrl string // the async URL of the flow
	flowName string // the name of the function
	reqId    string // the request id
	httpreq  *http.Request

	openFaasEventHandler
	openFaasLogger
}

// Logger
func (logger *openFaasLogger) Configure(flowName string, requestId string) {}
func (logger *openFaasLogger) Log(str string) {
	fmt.Print(str)
}

// EventHandler

func (eh *openFaasEventHandler) Configure(flowName string, requestId string) {
	eh.flowName = flowName
}

func (eh *openFaasEventHandler) Init() error {
	var err error
	// initialize traceserve if tracing enabled
	eh.tracer, err = initRequestTracer(eh.flowName)
	if err != nil {
		return fmt.Errorf("failed to init request tracer, error %v", err)
	}
	return nil
}

func (eh *openFaasEventHandler) ReportRequestStart(requestId string) {

}

func (eh *openFaasEventHandler) ReportRequestFailure(requestId string, err error) {

}

func (eh *openFaasEventHandler) ReportExecutionForward(currentNodeId string, requestId string) {
	eh.currentNodeId = currentNodeId
}

func (eh *openFaasEventHandler) ReportExecutionContinuation(requestId string) {
}

func (eh *openFaasEventHandler) ReportRequestEnd(requestId string) {
}

func (eh *openFaasEventHandler) ReportNodeStart(nodeId string, requestId string) {
	eh.tracer.startNodeSpan(nodeId, requestId)
}

func (eh *openFaasEventHandler) ReportNodeEnd(nodeId string, requestId string) {

}

func (eh *openFaasEventHandler) ReportNodeFailure(nodeId string, requestId string, err error) {

}

func (eh *openFaasEventHandler) ReportOperationStart(operationId string, nodeId string, requestId string) {
}

func (eh *openFaasEventHandler) ReportOperationEnd(operationId string, nodeId string, requestId string) {

}

func (eh *openFaasEventHandler) ReportOperationFailure(operationId string, nodeId string, requestId string, err error) {

}

func (eh *openFaasEventHandler) Flush() {
	eh.tracer.flushTracer()
}

// ExecutionHandler

func (of *openFaasExecutor) ExecuteOperation(operation sdk.Operation, data []byte) ([]byte, error) {
	var result []byte
	var err error

	ofoperation, ok := operation.(*faasflow.FaasOperation)
	if !ok {
		return nil, fmt.Errorf("Operation is not of type faasflow.OpenfaasOperation")
	}

	switch {
	// If function
	case ofoperation.Function != "":
		fmt.Printf("[Request `%s`] Executing function `%s`\n",
			of.reqId, ofoperation.Function)
		result, err = executeFunction(of.gateway, ofoperation, data)
		if err != nil {
			err = fmt.Errorf("Function(%s), error: function execution failed, %v",
				ofoperation.Function, err)
			if ofoperation.FailureHandler != nil {
				err = ofoperation.FailureHandler(err)
			}
			if err != nil {
				return nil, err
			}
		}
	// If callback
	case ofoperation.CallbackUrl != "":
		fmt.Printf("[Request `%s`] Executing callback `%s`\n",
			of.reqId, ofoperation.CallbackUrl)
		err = executeCallback(ofoperation, data)
		if err != nil {
			err = fmt.Errorf("Callback(%s), error: callback failed, %v",
				ofoperation.CallbackUrl, err)
			if ofoperation.FailureHandler != nil {
				err = ofoperation.FailureHandler(err)
			}
			if err != nil {
				return nil, err
			}
		}

	// If modifier
	default:
		fmt.Printf("[Request `%s`] Executing modifier\n", of.reqId)
		result, err = ofoperation.Mod(data)
		if err != nil {
			err = fmt.Errorf("error: Failed at modifier, %v", err)
			return nil, err
		}
		if result == nil {
			result = []byte("")
		}
	}

	return result, nil
}

// ExecutionRuntime

func (of *openFaasExecutor) RetriveExecutionState() []byte {
	// TODO:
	// get the execution state
	return nil
}

func (of *openFaasExecutor) RetriveNewRequest() (request *executor.RawRequest, err error) {
	// TODO:
	// get the raw request object
	return nil, nil
}

func (of *openFaasExecutor) ForwardExecutionState(state []byte) error {

	// build url for calling the flow in async
	httpreq, _ := http.NewRequest(http.MethodPost, of.asyncUrl, bytes.NewReader(state))
	httpreq.Header.Add("Accept", "application/json")
	httpreq.Header.Add("Content-Type", "application/json")
	httpreq.Header.Add("X-Faas-Flow-Reqid", of.reqId)

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

// Executor

func (of *openFaasExecutor) Configure(requestId string) {
	of.reqId = requestId
}

func (of *openFaasExecutor) GetFlowName() string {
	return of.flowName
}

func (of *openFaasExecutor) GetFlowDefinition(pipeline *sdk.Pipeline, context *sdk.Context) error {
	workflow := faasflow.GetWorkflow(pipeline)
	err := function.Define(workflow, context)
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
	of.asyncUrl = buildURL(of.gateway, "async-function", of.flowName)

	return nil
}

/*
func (of *openFaasExecutor) handleDagExport(req *HttpRequest) string {

	err := function.Define(fhandler.flow, context)
	if err != nil {
		panic(fmt.Sprintf("[Request `dag-export`] failed to export graph, error %v", err))
	}

	return fhandler.getPipeline().GetDagDefinition()
}*/

// RequestHandler

func (of *openFaasExecutor) Handle(req *HttpRequest, response *HttpResponse) error {

	err := of.init(req)
	if err != nil {
		panic(err.Error())
	}

	if isDagExportRequest(req) {
		// TODO:
		//result := of.handleDagExport(req)
		//response.Body = []byte(result)
	} else {

		var partial bool
		reqId := req.Header.Get("X-Faas-Flow-Reqid")
		if reqId == "" {
			partial = false
			// TODO
			// Create a executor.RawRequest object
			// set it on of
		} else {
			// TODO
			// set the partial data in the of
		}

		// Create a flow executor, openFaasExecutor implements executor
		flowExecutor := executor.CreateFlowExecutor(of)
		resp, err := flowExecutor.Execute(partial)
		if err != nil {
			panic(err.Error())
		}
		response.Body = resp
		response.Header.Set("X-Faas-Flow-Reqid", of.reqId)
	}

	response.StatusCode = http.StatusOK
	return nil
}
