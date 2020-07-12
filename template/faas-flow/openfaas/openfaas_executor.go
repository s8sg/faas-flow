package openfaas

import (
	"bytes"
	"fmt"
	"github.com/faasflow/runtime/controller/util"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	faasflow "github.com/faasflow/lib/openfaas"
	sdk "github.com/faasflow/sdk"
	"github.com/faasflow/sdk/executor"
	"handler/config"
	"handler/eventhandler"
	"handler/function"
	hlog "handler/log"
)

// A signature of SHA265 equivalent of github.com/s8sg/faas-flow
const defaultHmacKey = "71F1D3011F8E6160813B4997BA29856744375A7F26D427D491E1CCABD4627E7C"

// implements faasflow.Executor + RequestHandler
type OpenFaasExecutor struct {
	gateway      string
	asyncURL     string // the async URL of the flow
	flowName     string // the name of the function
	reqID        string // the request id
	CallbackURL  string // the callback url
	partialState []byte
	rawRequest   *executor.RawRequest
	StateStore   sdk.StateStore
	DataStore    sdk.DataStore
	EventHandler sdk.EventHandler
	logger       hlog.StdOutLogger
}

func (of *OpenFaasExecutor) HandleNextNode(partial *executor.PartialState) error {

	state, err := partial.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode partial state, error %v", err)
	}

	url, _ := url.Parse(of.asyncURL)
	url.Path = path.Join(url.Path, "flow", of.reqID, "forward")

	// build url for calling the flow in async
	httpReq, _ := http.NewRequest(http.MethodPost, url.String(), bytes.NewReader(state))
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add(util.RequestIdHeader, of.reqID)
	httpReq.Header.Set(util.CallbackUrlHeader, of.CallbackURL)

	fmt.Println(fmt.Sprint(of.asyncURL, of.reqID))
	fmt.Println(httpReq)

	// extend req span for async call
	if of.MonitoringEnabled() {
		faasHandler := of.EventHandler.(*eventhandler.FaasEventHandler)
		faasHandler.Tracer.ExtendReqSpan(of.reqID, faasHandler.CurrentNodeID, url.String(), httpReq)
	}

	client := &http.Client{}
	res, resErr := client.Do(httpReq)
	if resErr != nil {
		return resErr
	}

	defer res.Body.Close()
	resData, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("%d: %s", res.StatusCode, string(resData))
	}
	return nil
}

func (of *OpenFaasExecutor) GetExecutionOption(operation sdk.Operation) map[string]interface{} {
	options := make(map[string]interface{})
	options["gateway"] = of.gateway
	options["request-id"] = of.reqID

	return options
}

func (of *OpenFaasExecutor) HandleExecutionCompletion(data []byte) error {
	if of.CallbackURL == "" {
		return nil
	}

	log.Printf("calling callback url (%s) with result", of.CallbackURL)
	httpreq, _ := http.NewRequest(http.MethodPost, of.CallbackURL, bytes.NewReader(data))
	httpreq.Header.Add("X-Faas-Flow-ReqiD", of.reqID)
	client := &http.Client{}

	res, resErr := client.Do(httpreq)
	if resErr != nil {
		return resErr
	}
	defer res.Body.Close()
	resData, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to call callback %d: %s", res.StatusCode, string(resData))
	}

	return nil
}

func (of *OpenFaasExecutor) Configure(requestID string) {
	of.reqID = requestID
}

func (of *OpenFaasExecutor) GetFlowName() string {
	return of.flowName
}

func (of *OpenFaasExecutor) GetFlowDefinition(pipeline *sdk.Pipeline, context *sdk.Context) error {
	workflow := faasflow.GetWorkflow(pipeline)
	faasflowContext := (*faasflow.Context)(context)
	err := function.Define(workflow, faasflowContext)
	return err
}

func (of *OpenFaasExecutor) ReqValidationEnabled() bool {
	status := true
	hmacStatus := os.Getenv("validate_request")
	if strings.ToUpper(hmacStatus) == "FALSE" {
		status = false
	}
	return status
}

func (of *OpenFaasExecutor) GetValidationKey() (string, error) {
	key, keyErr := ReadSecret("faasflow-hmac-secret")
	if keyErr != nil {
		key = defaultHmacKey
	}
	return key, nil
}

func (of *OpenFaasExecutor) ReqAuthEnabled() bool {
	status := false
	verifyStatus := os.Getenv("authenticate_request")
	if strings.ToUpper(verifyStatus) == "TRUE" {
		status = true
	}
	return status
}

func (of *OpenFaasExecutor) GetReqAuthKey() (string, error) {
	key, keyErr := ReadSecret("faasflow-hmac-secret")
	return key, keyErr
}

func (of *OpenFaasExecutor) MonitoringEnabled() bool {
	tracing := os.Getenv("enable_tracing")
	if strings.ToUpper(tracing) == "TRUE" {
		return true
	}
	return false
}

func (of *OpenFaasExecutor) GetEventHandler() (sdk.EventHandler, error) {
	return of.EventHandler, nil
}

func (of *OpenFaasExecutor) LoggingEnabled() bool {
	return true
}

func (of *OpenFaasExecutor) GetLogger() (sdk.Logger, error) {
	return &of.logger, nil
}

func (of *OpenFaasExecutor) GetStateStore() (sdk.StateStore, error) {
	return of.StateStore, nil
}

func (of *OpenFaasExecutor) GetDataStore() (sdk.DataStore, error) {
	return of.DataStore, nil
}

func (of *OpenFaasExecutor) Init(req *http.Request) error {
	of.gateway = config.GatewayURL()
	of.flowName = getWorkflowNameFromHost(req.Host)
	if of.flowName == "" {
		return fmt.Errorf("failed to parse workflow name from host")
	}
	of.asyncURL = buildURL("http://"+of.gateway, "async-function", of.flowName)

	callbackURL := req.Header.Get("X-Faas-Flow-Callback-Url")
	of.CallbackURL = callbackURL

	faasHandler := of.EventHandler.(*eventhandler.FaasEventHandler)
	faasHandler.Header = req.Header

	return nil
}

// internal

var re = regexp.MustCompile(`(?m)^[^:.]+\s*`)

// getWorkflowNameFromHostFromHost returns the flow name from env
func getWorkflowNameFromHost(host string) string {
	matches := re.FindAllString(host, -1)
	if matches[0] != "" {
		return matches[0]
	}
	return ""
}

// buildURL builds execution url for the flow
func buildURL(gateway, rPath, function string) string {
	u, _ := url.Parse(gateway)
	u.Path = path.Join(u.Path, rPath, function)
	return u.String()
}
