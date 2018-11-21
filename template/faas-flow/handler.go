package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	hmac "github.com/alexellis/hmac"
	"github.com/rs/xid"
	faasflow "github.com/s8sg/faas-flow"
	sdk "github.com/s8sg/faas-flow/sdk"
	"handler/function"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

const (
	// A signature of SHA265 equivalent of "github.com/s8sg/faasflow"
	defaultHmacKey = "D9D98C7EBAA7267BCC4F0280FC5BA4273F361B00D422074985A41AE1338F1B61"
)

var (
	// flowName the name of the flow
	flowName = ""
)

type flowHandler struct {
	flow        *faasflow.Workflow // the faasflow
	id          string             // the request id
	query       string             // the query string by user
	asyncUrl    string             // the async URL of the flow
	pipelineDef []byte             // the pipeline definition
	partial     bool
	finished    bool // denots the flow has finished execution

	stateStore faasflow.StateStore // the state store
	dataStore  faasflow.DataStore  // the data store
}

// buildURL builds openfaas function execution url for the flow
func buildURL(gateway, rpath, flow string) string {
	u, _ := url.Parse(gateway)
	u.Path = path.Join(u.Path, rpath+"/"+flow)
	return u.String()
}

// newWorkflowHandler creates a new flow handler object
func newWorkflowHandler(gateway string, name string, id string,
	query string, dataStore faasflow.DataStore) *flowHandler {

	fhandler := &flowHandler{}

	flow := faasflow.NewFaasflow()

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
		gateway = "gateway:8080"
	}
	return gateway
}

// hmacEnabled check if hmac is enabled
// hmac is enabled by default
func hmacEnabled() bool {
	status := true
	hmacStatus := os.Getenv("enable_hmac")
	if strings.ToUpper(hmacStatus) == "FALSE" {
		status = false
	}
	return status
}

// verifyRequest check if request need to be verified
// request verification is disabled by default
func verifyRequest() bool {
	status := false
	verifyStatus := os.Getenv("verify_request")
	if strings.ToUpper(verifyStatus) == "TRUE" {
		status = true
	}
	return status
}

// getHmacKey returns the value of hmac secret key faasflow-hmac-secret
// if the key is not provided it uses default hamc key
func getHmacKey() string {
	key, keyErr := readSecret("faasflow-hmac-secret")
	if keyErr != nil {
		log.Printf("Failed to load faasflow-hmac-secret using default")
		key = defaultHmacKey
	}
	return key
}

// getWorkflowName returns the flow name from env
func getWorkflowName() string {
	return os.Getenv("workflow_name")
}

// useIntermediateStorage check if IntermidiateStorage is enabled
func useIntermediateStorage() bool {
	storage := os.Getenv("intermediate_storage")
	if strings.ToUpper(storage) == "TRUE" {
		return true
	}
	return false
}

// getPipeline returns the underline flow.pipeline object
func (fhandler *flowHandler) getPipeline() *sdk.Pipeline {
	return fhandler.flow.GetPipeline()
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

// buildFunctionRequest build upstream request for function
func buildFunctionRequest(function string, data []byte, params map[string][]string, headers map[string]string) *http.Request {
	url := "http://" + function + ":8080"
	queryString := makeQueryStringFromParam(params)
	if queryString != "" {
		url = url + queryString
	}

	var method string

	if method, ok := headers["method"]; !ok {
		method = os.Getenv("default-method")
		if method == "" {
			method = "POST"
		}
	}

	httpreq, _ := http.NewRequest(method, url, bytes.NewBuffer(data))

	for key, value := range headers {
		httpreq.Header.Set(key, value)
	}

	return httpreq
}

// buildCallbackRequest build upstream request for callback
func buildCallbackRequest(callbackUrl string, data []byte, params map[string][]string, headers map[string]string) *http.Request {
	url := callbackUrl
	queryString := makeQueryStringFromParam(params)
	if queryString != "" {
		url = url + queryString
	}

	var method string

	if method, ok := headers["method"]; !ok {
		method = os.Getenv("default-method")
		if method == "" {
			method = "POST"
		}
	}

	httpreq, _ := http.NewRequest(method, url, bytes.NewBuffer(data))

	for key, value := range headers {
		httpreq.Header.Set(key, value)
	}

	return httpreq
}

// createContext create a context from request handler
func createContext(fhandler *flowHandler) *faasflow.Context {
	context := faasflow.CreateContext(fhandler.id, fhandler.getPipeline().ExecutionPosition+1,
		flowName, fhandler.dataStore)
	context.Query, _ = url.ParseQuery(fhandler.query)

	return context
}

// buildWorkflow builds a flow and context from raw request or partial completed request
func buildWorkflow(data []byte) (fhandler *flowHandler, requestData []byte) {

	requestId := ""
	var validateErr error

	if hmacEnabled() {
		digest := os.Getenv("Http_X_Hub_Signature")
		key := getHmacKey()
		if len(digest) > 0 {
			validateErr = hmac.Validate(data, digest, key)
		} else {
			validateErr = fmt.Errorf("Http_X_Hub_Signature is not set")
		}
	}

	// decode the request to find if flow definition exists
	request, err := decodeRequest(data)
	switch {
	case err != nil:
		// New Request
		if verifyRequest() {
			if validateErr != nil {
				log.Fatalf("Failed to verify incoming request with Hmac, %v",
					validateErr.Error())
			} else {
				log.Printf("Incoming request verified successfully")
			}

		}

		// Generate request Id
		requestId = xid.New().String()
		log.Printf("[Request `%s`] Created", requestId)

		// create flow properties
		query := os.Getenv("Http_Query")
		dataStore := createDataStore()

		// Create fhandler
		fhandler = newWorkflowHandler("http://"+getGateway(), flowName,
			requestId, query, dataStore)
		fhandler.getPipeline().ExecutionPosition = 0

		// set request data
		requestData = data

		fhandler.partial = false

		// trace req - mark as start of req
		startReqSpan(requestId)
	default:
		// Partial Request
		// Get the request ID
		requestId = request.getID()
		log.Printf("[Request `%s`] Received", requestId)

		if hmacEnabled() {
			if validateErr != nil {
				log.Fatalf("[Request `%s`] Invalid Hmac, %v",
					requestId, validateErr.Error())
			} else {
				log.Printf("[Request `%s`] Valid Hmac", requestId)
			}
		}

		// get flow properties
		executionState := request.getExecutionState()
		query := request.getQuery()
		dataStore := retriveDataStore(request.getContextStore())

		// Create flow and apply execution state
		fhandler = newWorkflowHandler("http://"+getGateway(), flowName,
			requestId, query, dataStore)
		fhandler.getPipeline().ApplyState(executionState)

		// set request data
		requestData = request.getData()

		fhandler.partial = true

		// Continue request span
		continueReqSpan(requestId)
	}

	return
}

// executeFunction executes a function call
func executeFunction(pipeline *sdk.Pipeline, function *sdk.Function, data []byte) ([]byte, error) {
	var err error
	var result []byte

	name := function.GetName()
	params := function.GetParams()
	headers := function.GetHeaders()

	httpreq := buildFunctionRequest(name, data, params, headers)

	client := &http.Client{}
	resp, err := client.Do(httpreq)
	if err != nil {
		return []byte{}, err
	}

	if function.OnResphandler != nil {
		result, err = function.OnResphandler(resp)
	} else {
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			result, _ = ioutil.ReadAll(resp.Body)
			return result, err
		}

		result, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return result, err
		}
	}

	return result, err
}

// executeCallback executes a callback
func executeCallback(pipeline *sdk.Pipeline, function *sdk.Function, data []byte) error {
	var err error

	cburl := function.CallbackUrl
	params := function.GetParams()
	headers := function.GetHeaders()

	httpreq := buildCallbackRequest(cburl, data, params, headers)

	client := &http.Client{}
	resp, err := client.Do(httpreq)
	cbresult, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		err := fmt.Errorf("%v:%s", err, string(cbresult))
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		err := fmt.Errorf("%v:%s", err, string(cbresult))
		return err
	}
	return nil

}

// executePhase executes a phase on a faas chain
func executeChain(fhandler *flowHandler, request []byte) ([]byte, error) {
	var result []byte
	var err error

	pipeline := fhandler.getPipeline()
	phase := pipeline.GetCurrentPhase()

	log.Printf("[Request `%s`] Executing phase %d", fhandler.id, pipeline.ExecutionPosition)

	// trace phase - mark as start of phase
	startPhaseSpan(pipeline.ExecutionPosition, fhandler.id)

	// Execute all function
	for _, function := range phase.GetFunctions() {

		switch {
		// If function
		case function.GetName() != "":
			log.Printf("[Request `%s`] Executing function `%s`",
				fhandler.id, function.GetName())
			if result == nil {
				result, err = executeFunction(pipeline, function, request)
			} else {
				result, err = executeFunction(pipeline, function, result)
			}
			if err != nil {
				err = fmt.Errorf("Phase(%d), Function(%s), error: function execution failed, %v%s",
					pipeline.ExecutionPosition, function.GetName(), err, string(result))
				if function.FailureHandler != nil {
					err = function.FailureHandler(err)
				}
				if err != nil {
					stopPhaseSpan(pipeline.ExecutionPosition)
					return nil, err
				}
			}
		// If callback
		case function.CallbackUrl != "":
			log.Printf("[Request `%s`] Executing callback `%s`",
				fhandler.id, function.CallbackUrl)
			if result == nil {
				err = executeCallback(pipeline, function, request)
			} else {
				err = executeCallback(pipeline, function, result)
			}
			if err != nil {
				err = fmt.Errorf("Phase(%d), Callback(%s), error: callback failed, %v",
					pipeline.ExecutionPosition, function.CallbackUrl, err)
				if function.FailureHandler != nil {
					err = function.FailureHandler(err)
				}
				if err != nil {
					stopPhaseSpan(pipeline.ExecutionPosition)
					return nil, err
				}
			}

		// If modifier
		default:
			log.Printf("[Request `%s`] Executing modifier", fhandler.id)
			if result == nil {
				result, err = function.Mod(request)
			} else {
				result, err = function.Mod(result)
			}
			if err != nil {
				err = fmt.Errorf("Phase(%d), error: Failed at modifier, %v",
					pipeline.ExecutionPosition, err)
				return nil, err
			}
			if result == nil {
				result = []byte("")
			}
		}
	}

	log.Printf("[Request `%s`] Phase %d completed successfully", fhandler.id, pipeline.ExecutionPosition)

	// trace phase - mark as end of phase
	stopPhaseSpan(pipeline.ExecutionPosition)

	// Update execution position
	pipeline.UpdateExecutionPosition()

	return result, nil
}

// executeDag executes a node on a faas chain
func executeDag(fhandler *flowHandler, request []byte) ([]byte, error) {
	var result []byte
	var err error

	pipeline := fhandler.getPipeline()
	node := pipeline.GetCurrentNode()

	log.Printf("[Request `%s`] Executing node %s", fhandler.id, pipeline.DagExecutionPosition)

	// trace phase - mark as start of phase
	startNodeSpan(pipeline.DagExecutionPosition, fhandler.id)

	// Executes the node function
	function := node.Value()

	switch {
	// If function
	case function.GetName() != "":
		log.Printf("[Request `%s`] Executing function `%s`",
			fhandler.id, function.GetName())
		if result == nil {
			result, err = executeFunction(pipeline, function, request)
		} else {
			result, err = executeFunction(pipeline, function, result)
		}
		if err != nil {
			err = fmt.Errorf("Phase(%d), Function(%s), error: function execution failed, %v",
				pipeline.ExecutionPosition, function.GetName(), err)
			if function.FailureHandler != nil {
				err = function.FailureHandler(err)
			}
			if err != nil {
				stopPhaseSpan(pipeline.ExecutionPosition)
				return nil, err
			}
		}
	// If callback
	case function.CallbackUrl != "":
		log.Printf("[Request `%s`] Executing callback `%s`",
			fhandler.id, function.CallbackUrl)
		if result == nil {
			err = executeCallback(pipeline, function, request)
		} else {
			err = executeCallback(pipeline, function, result)
		}
		if err != nil {
			err = fmt.Errorf("Phase(%d), Callback(%s), error: callback failed, %v",
				pipeline.ExecutionPosition, function.CallbackUrl, err)
			if function.FailureHandler != nil {
				err = function.FailureHandler(err)
			}
			if err != nil {
				stopPhaseSpan(pipeline.ExecutionPosition)
				return nil, err
			}
		}

	// If modifier
	default:
		log.Printf("[Request `%s`] Executing modifier", fhandler.id)
		if result == nil {
			result, err = function.Mod(request)
		} else {
			result, err = function.Mod(result)
		}
		if err != nil {
			err = fmt.Errorf("Phase(%d), error: Failed at modifier, %v",
				pipeline.ExecutionPosition, err)
			return nil, err
		}
		if result == nil {
			result = []byte("")
		}
	}

	log.Printf("[Request `%s`] Node %s completed successfully", fhandler.id, pipeline.DagExecutionPosition)

	// trace phase - mark as end of phase
	stopNodeSpan(pipeline.DagExecutionPosition)

	// Update execution position
	//pipeline.UpdateExecutionPosition()

	return result, nil
}

// execute functions for current phase
func execute(fhandler *flowHandler, request []byte) ([]byte, error) {
	switch fhandler.getPipeline().PipelineType {
	case sdk.TYPE_CHAIN:
		return executeChain(fhandler, request)
	case sdk.TYPE_DAG:
		return executeDag(fhandler, request)
	}
	return nil, nil
}

// forwardAsync forward async request to faasflow
func forwardAsync(fhandler *flowHandler, result []byte) ([]byte, error) {
	var hash []byte
	store := make(map[string]string)

	// get pipeline
	pipeline := fhandler.getPipeline()

	// Get pipeline state
	pipelineState := pipeline.GetState()

	defaultStore, ok := fhandler.dataStore.(*requestEmbedDataStore)
	if ok {
		store = defaultStore.store
	}

	// Build request
	uprequest := buildRequest(fhandler.id, string(pipelineState), fhandler.query, result, store)

	// Make request data
	data, _ := uprequest.encode()

	// Check if HMAC used
	if hmacEnabled() {
		key := getHmacKey()
		hash = hmac.Sign(data, []byte(key))
	}

	// build url for calling the flow in async
	httpreq, _ := http.NewRequest(http.MethodPost, fhandler.asyncUrl, bytes.NewReader(data))
	httpreq.Header.Add("Accept", "application/json")
	httpreq.Header.Add("Content-Type", "application/json")

	// If hmac is enabled set digest
	if hmacEnabled() {
		httpreq.Header.Add("X-Hub-Signature", "sha1="+hex.EncodeToString(hash))
	}

	// extend req span for async call
	extendReqSpan(pipeline.ExecutionPosition-1, fhandler.asyncUrl, httpreq)

	client := &http.Client{}
	res, resErr := client.Do(httpreq)
	resdata, _ := ioutil.ReadAll(res.Body)
	if resErr != nil {
		return resdata, resErr
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		return resdata, fmt.Errorf(res.Status)
	}
	return resdata, nil
}

// handlePhaseResponse handle response of a pipeline of type chain
func handleChainResponse(fhandler *flowHandler, context *faasflow.Context, result []byte) ([]byte, error) {
	// get pipeline
	pipeline := fhandler.getPipeline()

	switch {

	// If pipeline has only one phase or if the pipeline has completed excution return
	case pipeline.CountPhases() == 1 || pipeline.GetCurrentPhase() == nil:
		fhandler.finished = true
		return result, nil

	// In default case we forward the partial flow to same flow-function
	default:
		// If intermediate storage is enabled
		if useIntermediateStorage() {
			key := fmt.Sprintf("intermediate-result-%d", pipeline.ExecutionPosition)
			serr := context.Set(key, result)
			if serr != nil {
				return []byte(""), fmt.Errorf("failed to store intermediate result, error %v", serr)
			}
			log.Printf("[Request `%s`] Intermidiate result for Phase %d stored as %s",
				fhandler.id, pipeline.ExecutionPosition, key)
			result = []byte("")
		}

		// forward the flow request
		resp, forwardErr := forwardAsync(fhandler, result)
		if forwardErr != nil {
			return nil, fmt.Errorf("Phase(%d): error: %v, %s, url %s",
				pipeline.ExecutionPosition, forwardErr,
				string(resp), fhandler.asyncUrl)
		}

		log.Printf("[Request `%s`] Async request submitted for Phase %d",
			fhandler.id, pipeline.ExecutionPosition)
	}

	return []byte(""), nil

}

// handlePhaseResponse handle response of a pipeline of type chain
func handleDagResponse(fhandler *flowHandler, context *faasflow.Context, result []byte) ([]byte, error) {

	// get pipeline
	pipeline := fhandler.getPipeline()

	switch {

	// Check if pipeline has only one node or if the pipeline has completed excution return
	case pipeline.CountNodes() == 1 || pipeline.GetNextNodes() == nil:
		fhandler.finished = true
		return result, nil

	default:
		// If intermediate storage is enabled
		if useIntermediateStorage() {
			key := fmt.Sprintf("intermediate-result-%s", pipeline.DagExecutionPosition)
			serr := context.Set(key, result)
			if serr != nil {
				return []byte(""), fmt.Errorf("failed to store intermediate result, error %v", serr)
			}
			log.Printf("[Request `%s`] Intermidiate result for Node %s stored as %s",
				fhandler.id, pipeline.DagExecutionPosition, key)
			result = []byte("")
		}

		// Check if pipeline is active
		if !isActive(fhandler) {
			return []byte(""), fmt.Errorf("flow has been terminated")
		}

		nodes := pipeline.GetNextNodes()
		for _, node := range nodes {
			// Get node Indegree
			inDegree := node.Indegree()
			// Update the state of indegree completion
			inDegreeUpdatedCount, err := fhandler.stateStore.IncrementCounter(node.Id)
			if err != nil {
				return []byte(""), fmt.Errorf("failed to update inDegree counter for node %s", node.Id)
			}
			// If all indegree has finished
			if inDegree == inDegreeUpdatedCount {
				// Set the DagExecutionPosition as of next Node
				pipeline.DagExecutionPosition = node.Id
				// forward the flow request
				resp, forwardErr := forwardAsync(fhandler, result)
				if forwardErr != nil {
					return nil, fmt.Errorf("Node(%s): error: %v, %s, url %s",
						node.Id, forwardErr,
						string(resp), fhandler.asyncUrl)
				}

				log.Printf("[Request `%s`] Async request submitted for Node %s",
					fhandler.id, node.Id)
			}
		}
		fhandler.finished = true
	}

	return []byte(""), nil
}

// isActive Cheks if pipeline is active
func isActive(fhandler *flowHandler) bool {

	state, err := fhandler.stateStore.GetState()
	if err != nil {
		log.Printf("[Request `%s`] Failed to obtain pipeline state", fhandler.id)
		return false
	}

	return state
}

// handleResponse Handle request Response for a faasflow perform response/asyncforward
func handleResponse(fhandler *flowHandler, context *faasflow.Context, result []byte) ([]byte, error) {

	switch fhandler.getPipeline().PipelineType {
	case sdk.TYPE_CHAIN:
		return handleChainResponse(fhandler, context, result)
	case sdk.TYPE_DAG:
		return handleDagResponse(fhandler, context, result)
	}
	return nil, nil
}

// handleFailure handles failure with failure handler and call finally
func handleFailure(fhandler *flowHandler, context *faasflow.Context, err error) {
	var data []byte

	context.State = faasflow.StateFailure
	// call failure handler if available
	if fhandler.getPipeline().FailureHandler != nil {
		log.Printf("[Request `%s`] Calling failure handler for error, %v",
			fhandler.id, err)
		data, err = fhandler.getPipeline().FailureHandler(err)
	}

	// If pipeline type is dag mark the state as failure
	if fhandler.getPipeline().PipelineType == sdk.TYPE_DAG {
		serr := fhandler.stateStore.SetState(false)
		if serr != nil {
			log.Printf("[Request `%s`] Failed to mark dag state, error %v", fhandler.id, serr)
		}
	}

	fhandler.finished = true

	// call finally handler if available
	if fhandler.getPipeline().Finally != nil {
		log.Printf("[Request `%s`] Calling Finally handler with state: %s",
			fhandler.id, faasflow.StateFailure)
		fhandler.getPipeline().Finally(faasflow.StateFailure)
	}
	if data != nil {
		fmt.Printf("%s", string(data))
	}

	if fhandler.stateStore != nil {
		fhandler.stateStore.Cleanup()
	}
	fhandler.dataStore.Cleanup()

	// stop req span if request has finished
	stopReqSpan()

	// flash any pending trace item if tracing enabled
	flushTracer()

	log.Fatalf("[Request `%s`] Failed, %v", fhandler.id, err)
}

// getIntermediateData gets the intermidiate data if intermidiateStorage is enabled
func getIntermediateData(fhandler *flowHandler, context *faasflow.Context, data []byte) ([]byte, error) {
	switch fhandler.getPipeline().PipelineType {
	case sdk.TYPE_CHAIN:
		// For partially completed requests
		// If IntermidiateStorage is enabled get the data from dataStoretore
		if fhandler.getPipeline().ExecutionPosition != 0 {
			key := fmt.Sprintf("intermediate-result-%d", fhandler.getPipeline().ExecutionPosition)
			idata, gerr := context.GetBytes(key)
			if gerr != nil {
				return data, gerr
			}
			log.Printf("[Request `%s`] Intermidiate result for Phase %d retrived from %s",
				fhandler.id, fhandler.getPipeline().ExecutionPosition, key)
			context.Del(key)
			data = idata
		}

	case sdk.TYPE_DAG:
		// For partially completed requests
		// If IntermidiateStorage is enabled get the data from dataStoretore
		if !fhandler.getPipeline().IsInitialNode() {
			key := fmt.Sprintf("intermediate-result-%s", fhandler.getPipeline().DagExecutionPosition)
			idata, gerr := context.GetBytes(key)
			if gerr != nil {
				return data, gerr
			}
			log.Printf("[Request `%s`] Intermidiate result for Node %s retrived from %s",
				fhandler.id, fhandler.getPipeline().DagExecutionPosition, key)
			context.Del(key)
			data = idata
		}

	}
	return data, nil
}

// initializeStore initialize the store and return true if both store if overriden
func initializeStore(fhandler *flowHandler) (bool, error) {

	stateSDefined := false
	dataSOverride := false

	// Initialize the stateS
	stateS, err := function.DefineStateStore()
	if err != nil {
		return false, err
	}
	if stateS != nil {
		fhandler.stateStore = stateS
		stateSDefined = true
		fhandler.stateStore.Configure(flowName, fhandler.id)
		// If request is not partial initialize the stateStore
		if !fhandler.partial {
			err = fhandler.stateStore.Init()
			if err != nil {
				return false, err
			}
		}
	}

	// Initialize the dataS
	dataS, err := function.DefineDataStore()
	if err != nil {
		return false, err
	}
	if dataS != nil {
		fhandler.dataStore = dataS
		dataSOverride = true
	}
	fhandler.dataStore.Configure(flowName, fhandler.id)
	// If request is not partial initialize the dataStore
	if !fhandler.partial {
		err = fhandler.dataStore.Init()
		if err != nil {
			return false, err
		}
	}

	if stateSDefined && dataSOverride {
		return true, nil
	}

	return false, nil
}

// handleWorkflow handle the flow
func handleWorkflow(data []byte) string {

	var resp []byte
	var gerr error

	// Get flow name
	flowName = getWorkflowName()
	if flowName == "" {
		log.Fatalf("Error: flow name must be provided when deployed as function")
	}

	// initialize traceserve if tracing enabled
	err := initGlobalTracer(flowName)
	if err != nil {
		log.Printf(err.Error())
	}

	// Chain Execution Steps
	{
		// BUILD: build the flow based on execution request
		fhandler, data := buildWorkflow(data)

		// INIT STORE: Get definition of StateStore and DataStore
		storeOverride, err := initializeStore(fhandler)
		if err != nil {
			log.Fatalf("[Request `%s`] Failed to init flow, %v", fhandler.id, err)
		}

		// MAKE CONTEXT: make the request context from flow
		context := createContext(fhandler)

		// DEFINE: Get Pipeline definition from user implemented Define()
		err = function.Define(fhandler.flow, context)
		if err != nil {
			log.Fatalf("[Request `%s`] Failed to define flow, %v", fhandler.id, err)
		}

		// In case of DAG Store need to be external
		if fhandler.getPipeline().PipelineType == sdk.TYPE_DAG && !storeOverride {
			log.Fatalf("[Request `%s`] DAG flow need external State and Data Store", fhandler.id)
		}

		// For a new dag pipeline Create the vertex in
		if fhandler.getPipeline().PipelineType == sdk.TYPE_DAG && !fhandler.partial {
			err = fhandler.stateStore.Create(fhandler.getPipeline().GetAllNodesId())
			if err != nil {
				log.Fatalf("[Request `%s`] DAG state can not be initiated at StateStore, %v", fhandler.id, err)
			}
			serr := fhandler.stateStore.SetState(true)
			if serr != nil {
				log.Fatalf("[Request `%s`] Failed to mark dag state, error %v", fhandler.id, serr)
			}
		}

		// GETDATA: Get intermediate data from data store if not using default
		if useIntermediateStorage() {
			data, gerr = getIntermediateData(fhandler, context, data)
			if gerr != nil {
				gerr := fmt.Errorf("failed to retrive intermediate result, error %v", gerr)
				handleFailure(fhandler, context, gerr)
			}
		}

		// EXECUTE: execute the flow based on current phase
		result, err := execute(fhandler, data)
		if err != nil {
			handleFailure(fhandler, context, err)
		}

		// HANDLE: Handle the execution state and perform response/asyncforward
		resp, err = handleResponse(fhandler, context, result)
		if err != nil {
			handleFailure(fhandler, context, err)
		}

		if fhandler.finished {
			log.Printf("[Request `%s`] Completed successfully", fhandler.id)
			context.State = faasflow.StateSuccess
			if fhandler.getPipeline().Finally != nil {
				log.Printf("[Request `%s`] Calling Finally handler with state: %s",
					fhandler.id, faasflow.StateSuccess)
				fhandler.getPipeline().Finally(faasflow.StateSuccess)
			}
			if fhandler.stateStore != nil {
				fhandler.stateStore.Cleanup()
			}
			fhandler.dataStore.Cleanup()
		}
	}

	// stop req span if request has finished
	stopReqSpan()

	// flash any pending trace item if tracing enabled
	flushTracer()

	return string(resp)
}
