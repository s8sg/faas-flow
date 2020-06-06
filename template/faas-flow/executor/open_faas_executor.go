package executor

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"handler/config"
	"handler/function"
	hlog "handler/log"
	"handler/openfaas"

	faasflow "github.com/s8sg/faas-flow"
	"github.com/s8sg/faas-flow/sdk"
	"github.com/s8sg/faas-flow/sdk/executor"
)

// A signature of SHA265 equivalent of "github.com/s8sg/faas-flow"
const defaultHmacKey = "71F1D3011F8E6160813B4997BA29856744375A7F26D427D491E1CCABD4627E7C"

// implements faasflow.Executor + RequestHandler
type openFaasExecutor struct {
	gateway      string
	asyncURL     string // the async URL of the flow
	flowName     string // the name of the function
	reqID        string // the request id
	callbackURL  string // the callback url
	partialState []byte
	rawRequest   *executor.RawRequest
	stateStore   sdk.StateStore
	dataStore    sdk.DataStore
	logger       hlog.StdOutLogger

	openFaasEventHandler
}

func (of *openFaasExecutor) HandleNextNode(partial *executor.PartialState) error {

	state, err := partial.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode partial state, error %v", err)
	}

	// build url for calling the flow in async
	httpreq, _ := http.NewRequest(http.MethodPost, of.asyncURL, bytes.NewReader(state))
	httpreq.Header.Add("Accept", "application/json")
	httpreq.Header.Add("Content-Type", "application/json")
	httpreq.Header.Add("X-Faas-Flow-Reqid", of.reqID)
	httpreq.Header.Set("X-Faas-Flow-State", "partial")
	httpreq.Header.Set("X-Faas-Flow-Callback-Url", of.callbackURL)

	// extend req span for async call
	if of.MonitoringEnabled() {
		of.tracer.extendReqSpan(of.reqID, of.openFaasEventHandler.currentNodeID,
			of.asyncURL, httpreq)
	}

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
	options["request-id"] = of.reqID

	return options
}

func (of *openFaasExecutor) HandleExecutionCompletion(data []byte) error {
	if of.callbackURL == "" {
		return nil
	}

	log.Printf("calling callback url (%s) with result", of.callbackURL)
	httpreq, _ := http.NewRequest(http.MethodPost, of.callbackURL, bytes.NewReader(data))
	httpreq.Header.Add("X-Faas-Flow-ReqiD", of.reqID)
	client := &http.Client{}

	res, resErr := client.Do(httpreq)
	if resErr != nil {
		return resErr
	}
	defer res.Body.Close()
	resdata, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to call callback %d: %s", res.StatusCode, string(resdata))
	}

	return nil
}

func (of *openFaasExecutor) Configure(requestID string) {
	of.reqID = requestID
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
	key, keyErr := openfaas.ReadSecret("faasflow-hmac-secret")
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
	key, keyErr := openfaas.ReadSecret("faasflow-hmac-secret")
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
	return &of.logger, nil
}

func (of *openFaasExecutor) GetStateStore() (sdk.StateStore, error) {
	return of.stateStore, nil
}

func (of *openFaasExecutor) GetDataStore() (sdk.DataStore, error) {
	return of.dataStore, nil
}

// internal

func (of *openFaasExecutor) init(req *http.Request) error {
	of.gateway = config.GatewayURL()
	of.flowName = getWorkflowNameFromHost(req.Host)
	if of.flowName == "" {
		return fmt.Errorf("failed to parse workflow name from host")
	}
	of.asyncURL = buildURL("http://"+of.gateway, "async-function", of.flowName)

	if of.MonitoringEnabled() {
		var err error
		// initialize trace server if tracing enabled
		of.openFaasEventHandler.tracer, err = initRequestTracer(of.flowName)
		if err != nil {
			return fmt.Errorf("failed to init request tracer, error %v", err)
		}
	}

	return nil
}
