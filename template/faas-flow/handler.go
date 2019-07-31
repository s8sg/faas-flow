package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"handler/function"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	xid "github.com/rs/xid"
	faasflow "github.com/s8sg/faas-flow"
	sdk "github.com/s8sg/faas-flow/sdk"
)

const (
	// A signature of SHA265 equivalent of "github.com/s8sg/faas-flow"
	defaultHmacKey          = "71F1D3011F8E6160813B4997BA29856744375A7F26D427D491E1CCABD4627E7C"
	counterUpdateRetryCount = 10
)

var (
	// flowName the name of the flow
	flowName = ""
	re       = regexp.MustCompile(`(?m)^[^:.]+\s*`)
)

type openFaasExecutor struct {
	asyncUrl string // the async URL of the flow
}

// buildURL builds openfaas function execution url for the flow
func buildURL(gateway, rpath, function string) string {
	u, _ := url.Parse(gateway)
	u.Path = path.Join(u.Path, rpath+"/"+function)
	return u.String()
}

// newWorkflowHandler creates a new flow handler object
func newWorkflowHandler(gateway string, name string, id string,
	query string, dataStore faasflow.DataStore) *flowHandler {

	fhandler := &flowHandler{}

	flow := faasflow.GetWorkflow()

	fhandler.flow = flow

	fhandler.asyncUrl = buildURL(gateway, "async-function", name)

	fhandler.id = id
	fhandler.query = query
	fhandler.dataStore = dataStore

	return fhandler
}

// readSecret reads a secret from /var/openfaas/secrets or from
// env-var 'secret_mount_path' if set.
func readSecret(key string) (string, error) {
	basePath := "/var/openfaas/secrets/"
	if len(os.Getenv("secret_mount_path")) > 0 {
		basePath = os.Getenv("secret_mount_path")
	}

	readPath := path.Join(basePath, key)
	secretBytes, readErr := ioutil.ReadFile(readPath)
	if readErr != nil {
		return "", fmt.Errorf("unable to read secret: %s, error: %s", readPath, readErr)
	}
	val := strings.TrimSpace(string(secretBytes))
	return val, nil
}

// getGateway return the gateway address from env
func getGateway() string {
	gateway := os.Getenv("gateway")
	if gateway == "" {
		gateway = "gateway.openfaas:8080"
	}
	return gateway
}

// getWorkflowNameFromHostFromHost returns the flow name from env
func getWorkflowNameFromHost(host string) string {
	matches := re.FindAllString(host, -1)
	if matches[0] != "" {
		return matches[0]
	}
	return ""
}

// makeQueryStringFromParam create query string from provided query
func makeQueryStringFromParam(params map[string][]string) string {
	if params == nil {
		return ""
	}
	result := ""
	for key, array := range params {
		for _, value := range array {
			keyVal := fmt.Sprintf("%s-%s", key, value)
			if result == "" {
				result = "?" + keyVal
			} else {
				result = result + "&" + keyVal
			}
		}
	}
	return result
}

// buildHttpRequest build upstream request for function
func buildHttpRequest(url string, method string, data []byte, params map[string][]string,
	headers map[string]string) (*http.Request, error) {

	queryString := makeQueryStringFromParam(params)
	if queryString != "" {
		url = url + queryString
	}

	httpreq, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		httpreq.Header.Add(key, value)
	}

	return httpreq, nil
}

// ReportNodeStart report start of a node execution
func (of *openFaasExecutor) ReportNodeStart(nodeId string, requestId string) {
	of.tracer.startNodeSpan(currentNode.GetUniqueId(), fhandler.id)
}

// ReportNodeEnd reports end of a node execution
func (of *openFaasExecutor) ReportNodeEnd(nodeId string, requestId string) {

}

// ReportNodeFailure reports failure of a node execution
func (of *openFaasExecutor) ReportNodeFailure(nodeId string, requestId string, err error) {

}

// ReportRequestStart reports start of a request
func (of *openFaasExecutor) ReportRequestStart(requestId string) {

}

// ReportRequestFailure reports failure of a request
func (of *openFaasExecutor) ReportRequestFailure(requestId string, err error) {

}

// ReportRequestEnd reports end of a failure
func (of *openFaasExecutor) ReportRequestEnd(requestId string) {

}

// Flush flush all reports
func (of *openFaasExecutor) Flush() {
	of.tracer.flushTracer()
}

// IsReqValidationEnabled check if request validation is enabled
// validation is enabled by default
func (of *openFaasExecutor) ReqValidationEnabled() bool {
	status := true
	hmacStatus := os.Getenv("validate_request")
	if strings.ToUpper(hmacStatus) == "FALSE" {
		status = false
	}
	return status
}

// IsReqAuthEnabled check if request need to be authenticated
// request authentication is disabled by default
func (of *openFaasExecutor) ReqAuthEnabled() bool {
	status := false
	verifyStatus := os.Getenv("authenticate_request")
	if strings.ToUpper(verifyStatus) == "TRUE" {
		status = true
	}
	return status
}

// GetValidationKey returns the value of hmac secret key faasflow-hmac-secret
// if the key is not provided it uses default hamc key
func (of *openFaasExecutor) GetValidationKey() string {
	key, keyErr := readSecret("faasflow-hmac-secret")
	if keyErr != nil {
		key = defaultHmacKey
	}
	return key
}

// GetReqAuthKey  returns the value of hmac secret key faasflow-hmac-secret
// if the key is not provided it uses default hamc key
func (of *openFaasExecutor) GetReqAuthKey() string {
	key, keyErr := readSecret("faasflow-hmac-secret")
	if keyErr != nil {
		key = defaultHmacKey
	}
	return key
}

// GetFlowName returns the flow name
func (of *openFaasExecutor) GetFlowName() string {
	return flowName
}

// createContext create a context from request handler
func createContext(fhandler *flowHandler) *faasflow.Context {
	context := faasflow.CreateContext(fhandler.id, "",
		flowName, fhandler.dataStore)
	context.Query, _ = url.ParseQuery(fhandler.query)

	return context
}

// executeFunction executes a function call
func executeFunction(pipeline *sdk.Pipeline, operation *sdk.Operation, data []byte) ([]byte, error) {
	var err error
	var result []byte

	name := operation.Function
	params := operation.GetParams()
	headers := operation.GetHeaders()

	gateway := getGateway()
	url := buildURL("http://"+gateway, "function", name)

	method := os.Getenv("default-method")
	if method == "" {
		method = "POST"
	}

	if m, ok := headers["method"]; ok {
		method = m
	}

	httpreq, err := buildHttpRequest(url, method, data, params, headers)
	if err != nil {
		return []byte{}, fmt.Errorf("cannot connect to Function on URL: %s", url)
	}

	if operation.Requesthandler != nil {
		operation.Requesthandler(httpreq)
	}

	client := &http.Client{}
	resp, err := client.Do(httpreq)
	if err != nil {
		return []byte{}, err
	}

	defer resp.Body.Close()
	if operation.OnResphandler != nil {
		result, err = operation.OnResphandler(resp)
	} else {
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			err = fmt.Errorf("invalid return status %d while connecting %s", resp.StatusCode, url)
			result, _ = ioutil.ReadAll(resp.Body)
		} else {
			result, err = ioutil.ReadAll(resp.Body)
		}
	}

	return result, err
}

// executeCallback executes a callback
func executeCallback(pipeline *sdk.Pipeline, operation *sdk.Operation, data []byte) error {
	var err error

	cburl := operation.CallbackUrl
	params := operation.GetParams()
	headers := operation.GetHeaders()

	method := os.Getenv("default-method")
	if method == "" {
		method = "POST"
	}

	if m, ok := headers["method"]; ok {
		method = m
	}

	httpreq, err := buildHttpRequest(cburl, method, data, params, headers)
	if err != nil {
		return fmt.Errorf("cannot connect to Function on URL: %s", cburl)
	}

	if operation.Requesthandler != nil {
		operation.Requesthandler(httpreq)
	}

	client := &http.Client{}
	resp, err := client.Do(httpreq)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if operation.OnResphandler != nil {
		_, err = operation.OnResphandler(resp)
	} else {
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			cbresult, _ := ioutil.ReadAll(resp.Body)
			err := fmt.Errorf("%v:%s", err, string(cbresult))
			return err
		}
	}
	return err

}

func (of *openFaasExecutor) ReportExecutionForward(currentNodeId string, requestId string) {
	fhandler.tracer.extendReqSpan(requestId, currentNodeId,
		fhandler.asyncUrl, httpreq)
}

// ExecuteOperation executes a operation for OpenFaaS faasflow
func (of *openFaasExecutor) ExecuteOperation(operation *sdk.Operation, data []byte) ([]byte, error) {
	var result []byte
	var err error

	ofoperation, ok = operation.(faasflow.OpenfaasOperation)
	if !ok {
		return nil, fmt.Errorf("Operation is not of type faasflow.OpenfaasOperation")
	}

	switch {
	// If function
	case ofoperation.Function != "":
		fmt.Printf("[Request `%s`] Executing function `%s`\n",
			fhandler.id, ofoperation.Function)
		result, err = executeFunction(ofoperation, data)
		if err != nil {
			err = fmt.Errorf("Node(%s), Function(%s), error: function execution failed, %v",
				currentNode.GetUniqueId(), ofoperation.Function, err)
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
			fhandler.id, ofoperation.CallbackUrl)
		result, err = executeCallback(ofoperation, data)
		if err != nil {
			err = fmt.Errorf("Node(%s), Callback(%s), error: callback failed, %v",
				currentNode.GetUniqueId(), ofoperation.CallbackUrl, err)
			if ofoperation.FailureHandler != nil {
				err = ofoperation.FailureHandler(err)
			}
			if err != nil {
				return nil, err
			}
		}

	// If modifier
	default:
		fmt.Printf("[Request `%s`] Executing modifier\n", fhandler.id)
		if result == nil {
			result, err = ofoperation.Mod(request)
		} else {
			result, err = ofoperation.Mod(result)
		}
		if err != nil {
			err = fmt.Errorf("Node(%s), error: Failed at modifier, %v",
				currentNode.GetUniqueId(), err)
			return nil, err
		}
		if result == nil {
			result = []byte("")
		}
	}

	return result, nil
}

// ForwardEncodedState forwards state for next stage of execution to faasflow
func (of *openFaasExecutor) ForwardEncodedState(state []byte) error {

	// build url for calling the flow in async
	httpreq, _ := http.NewRequest(http.MethodPost, of.asyncUrl, bytes.NewReader(state))
	httpreq.Header.Add("Accept", "application/json")
	httpreq.Header.Add("Content-Type", "application/json")

	// extend req span for async call (TODO : Get the value)
	fhandler.tracer.extendReqSpan(fhandler.id, currentNodeId,
		fhandler.asyncUrl, httpreq)

	client := &http.Client{}
	res, resErr := client.Do(httpreq)
	if resErr != nil {
		return nil, resErr
	}

	defer res.Body.Close()
	resdata, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("%d: %s", res.Status, string(resdata))
	}
	return nil
}

func (of *openFaasExecutor) GetStateStore() (faasflow.StateStore, error) {
	return function.DefineStateStore()
}

func (of *openFaasExecutor) GetDataStore() (faasflow.StateStore, error) {
	return function.DefineDataStore()
}

func (of *openFaasExecutor) GetFlowDefinition(workflow *faasflow.Workflow, context *faasflow.Context) {
	return function.Define(workflow, context)
}

// handleWorkflow handle the flow
func handleWorkflow(req *HttpRequest) (string, string) {

	data := req.Body
	queryString := req.QueryString
	header := req.Header

	of := &openFaasExecutor{}
	executor := CreateExecutor(of)
	resp, err := executor.Execute()
	if err != nil {
		panic(err)
	}

	return resp, executor.GetReqId()
}

// isDagExportRequest check if dag export request
func isDagExportRequest(req *HttpRequest) bool {
	values, err := url.ParseQuery(req.QueryString)
	if err != nil {
		return false
	}

	if strings.ToUpper(values.Get("export-dag")) == "TRUE" {
		return true
	}
	return false
}

// Handle handles the request
func (of *openFaasExecutor) Handle(req *HttpRequest, response *HttpResponse) (err error) {

	var result string
	var reqId string

	status := http.StatusOK

	// Get flow name
	if flowName == "" {
		flowName = getWorkflowNameFromHost(req.Host)
		if flowName == "" {
			panic(fmt.Sprintf("Error: workflow_name must be provided, specify workflow_name: <fucntion_name> using environment"))
		}
	}

	if isDagExportRequest(req) {
		result = handleDagExport(req)
	} else {
		result, reqId = handleWorkflow(req)
		response.Header.Set("X-Faas-Flow-Reqid", reqId)
	}

	response.Body = []byte(result)
	response.StatusCode = status

	return
}
