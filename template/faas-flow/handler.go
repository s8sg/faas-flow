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
	// A signature of SHA265 equivalent of "github.com/s8sg/faas-flow"
	defaultHmacKey = "71F1D3011F8E6160813B4997BA29856744375A7F26D427D491E1CCABD4627E7C"
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
	partial     bool               // denotes the flow is in partial execution state
	finished    bool               // denots the flow has finished execution

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

	flow := faasflow.NewFaasflow(name)

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
	// TODO: Provide Context
	context := faasflow.CreateContext(fhandler.id, "",
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

		// set request data
		requestData = data

		// Mark partial request
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

		// Mark partial request
		fhandler.partial = true

		// Continue request span
		continueReqSpan(requestId)
	}

	return
}

// executeFunction executes a function call
func executeFunction(pipeline *sdk.Pipeline, operation *sdk.Operation, data []byte) ([]byte, error) {
	var err error
	var result []byte

	name := operation.Function
	params := operation.GetParams()
	headers := operation.GetHeaders()

	httpreq := buildFunctionRequest(name, data, params, headers)

	client := &http.Client{}
	resp, err := client.Do(httpreq)
	if err != nil {
		return []byte{}, err
	}

	if operation.OnResphandler != nil {
		result, err = operation.OnResphandler(resp)
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
func executeCallback(pipeline *sdk.Pipeline, operation *sdk.Operation, data []byte) error {
	var err error

	cburl := operation.CallbackUrl
	params := operation.GetParams()
	headers := operation.GetHeaders()

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

// execute executes a node on a faas-flow dag
func execute(fhandler *flowHandler, request []byte) ([]byte, error) {
	var result []byte
	var err error

	pipeline := fhandler.getPipeline()

	currentNode, currentDag := pipeline.GetCurrentNodeDag()

	log.Printf("[Request `%s`] Executing node %s", fhandler.id, currentNode.GetUniqueId())

	// recurse to the subdag - if a node is dynamic stop to evaluate it
	for true {
		if currentNode.Dynamic() {
			return request, nil
		}
		subdag := currentNode.SubDag()
		if subdag == nil {
			break
		}
		currentDag = subdag
		currentNode = currentDag.GetInitialNode()
		log.Printf("[Request `%s`] Executing node %s", fhandler.id, currentNode.GetUniqueId())
		pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_INCREMENT, currentNode.Id)
	}

	// trace node - mark as start of node
	startNodeSpan(currentNode.GetUniqueId(), fhandler.id)

	// Execute all operation
	for _, operation := range currentNode.Operations() {

		switch {
		// If function
		case operation.Function != "":
			log.Printf("[Request `%s`] Executing function `%s`",
				fhandler.id, operation.Function)
			if result == nil {
				result, err = executeFunction(pipeline, operation, request)
			} else {
				result, err = executeFunction(pipeline, operation, result)
			}
			if err != nil {
				err = fmt.Errorf("Node(%s), Function(%s), error: function execution failed, %v",
					currentNode.GetUniqueId(), operation.Function, err)
				if operation.FailureHandler != nil {
					err = operation.FailureHandler(err)
				}
				if err != nil {
					stopNodeSpan(currentNode.GetUniqueId())
					return nil, err
				}
			}
		// If callback
		case operation.CallbackUrl != "":
			log.Printf("[Request `%s`] Executing callback `%s`",
				fhandler.id, operation.CallbackUrl)
			if result == nil {
				err = executeCallback(pipeline, operation, request)
			} else {
				err = executeCallback(pipeline, operation, result)
			}
			if err != nil {
				err = fmt.Errorf("Node(%s), Callback(%s), error: callback failed, %v",
					currentNode.GetUniqueId(), operation.CallbackUrl, err)
				if operation.FailureHandler != nil {
					err = operation.FailureHandler(err)
				}
				if err != nil {
					stopNodeSpan(currentNode.GetUniqueId())
					return nil, err
				}
			}

		// If modifier
		default:
			log.Printf("[Request `%s`] Executing modifier", fhandler.id)
			if result == nil {
				result, err = operation.Mod(request)
			} else {
				result, err = operation.Mod(result)
			}
			if err != nil {
				stopNodeSpan(currentNode.GetUniqueId())
				err = fmt.Errorf("Node(%s), error: Failed at modifier, %v",
					currentNode.GetUniqueId(), err)
				return nil, err
			}
			if result == nil {
				result = []byte("")
			}
		}
	}

	log.Printf("[Request `%s`] Node %s completed successfully", fhandler.id, currentNode.GetUniqueId())

	// trace node - mark as end of node
	stopNodeSpan(currentNode.GetUniqueId())

	return result, nil
}

// forwardAsync forward async request to faasflow
func forwardAsync(fhandler *flowHandler, currentNodeId string, result []byte) ([]byte, error) {
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

	// extend req span for async call (TODO : Get the value)
	extendReqSpan(currentNodeId, fhandler.asyncUrl, httpreq)

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

func handleDynamicNode(fhandler *flowHandler, context *faasflow.Context, result []byte) ([]byte, error) {
	// get pipeline
	pipeline := fhandler.getPipeline()

	currentNode, _ := pipeline.GetCurrentNodeDag()

	currentNodeUniqueId := currentNode.GetUniqueId()

	log.Printf("[Request `%s`] Processing dynamic node %s", fhandler.id, currentNodeUniqueId)

	// subresults and subdags
	subresults := make(map[string][]byte)
	subdags := make(map[string]*sdk.Dag)
	options := []string{}

	condition := currentNode.GetCondition()
	foreach := currentNode.GetForEach()

	dependencyCount := 0

	switch {
	case condition != nil:
		conditions := condition(result)
		for _, conditionKey := range conditions {
			subdags[conditionKey] = currentNode.GetConditionalDag(conditionKey)
			subresults[conditionKey] = result
			options = append(options, conditionKey)
			dependencyCount = dependencyCount + 1
		}
	case foreach != nil:
		foreachResults := foreach(result)
		for foreachKey, foreachResult := range foreachResults {
			subdags[foreachKey] = currentNode.SubDag()
			subresults[foreachKey] = foreachResult
			options = append(options, foreachKey)
			dependencyCount = dependencyCount + 1
		}
	}

	if dependencyCount == 0 {
		return nil, fmt.Errorf("[Request `%s`] Dynamic Node %s, failed to execute as condition/foreach returned no option",
			fhandler.id, currentNodeUniqueId)
	}

	// Set the no of depedency graph
	pipeline.DynamicDependencyCount[currentNodeUniqueId] = dependencyCount
	pipeline.AllDynamicOption[currentNodeUniqueId] = options

	for dynamicOption, subdag := range subdags {

		subNode := subdag.GetInitialNode()
		intermediateData := subresults[dynamicOption]

		err := fhandler.stateStore.Create(subdag.GetNodes(dynamicOption))
		if err != nil {
			delete(pipeline.DynamicDependencyCount, currentNodeUniqueId)
			delete(pipeline.AllDynamicOption, currentNodeUniqueId)
			// NOTE: This error will happen for the first time,
			// we dont perform depth decrement
			return nil, fmt.Errorf("[Request `%s`] Dynamic DAG %s-%s state can not be initiated at StateStore, %v",
				fhandler.id, subdag.Id, dynamicOption, err)
		}

		// If intermediate storage is enabled store data to intermediate storage
		if useIntermediateStorage() {
			// <option>-<currentnodeid>-<subnodeid>
			key := fmt.Sprintf("%s-%s-%s", dynamicOption, currentNodeUniqueId, subNode.GetUniqueId())
			serr := context.Set(key, intermediateData)
			if serr != nil {
				return []byte(""), fmt.Errorf("failed to store intermediate result, error %v", serr)
			}
			log.Printf("[Request `%s`] Intermidiate result from Node %s to %s stored as %s",
				fhandler.id, currentNodeUniqueId, subNode.GetUniqueId(), key)

			// intermediateData is set to blank once its stored in storage
			intermediateData = []byte("")
		}

		pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_INCREMENT, subNode.Id)
		pipeline.CurrentDynamicOption[currentNode.GetUniqueId()] = dynamicOption

		// forward the flow request
		resp, forwardErr := forwardAsync(fhandler, currentNodeUniqueId, intermediateData)
		if forwardErr != nil {
			// reset dag execution position
			pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_DECREMENT, currentNode.Id)
			delete(pipeline.DynamicDependencyCount, currentNodeUniqueId)
			delete(pipeline.AllDynamicOption, currentNodeUniqueId)
			return nil, fmt.Errorf("Node(%s): error: %v, %s, url %s",
				currentNodeUniqueId, forwardErr,
				string(resp), fhandler.asyncUrl)
		}

		log.Printf("[Request `%s`] Async request submitted for Node %s option %s",
			fhandler.id, currentNodeUniqueId, dynamicOption)

		delete(pipeline.CurrentDynamicOption, currentNodeUniqueId)
	}

	// reset pipeline
	if dependencyCount > 0 {
		pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_DECREMENT, currentNode.Id)
	}
	delete(pipeline.DynamicDependencyCount, currentNodeUniqueId)
	delete(pipeline.AllDynamicOption, currentNodeUniqueId)

	return []byte(""), nil

}

// handleResponse Handle request Response for a faasflow perform response/asyncforward
func handleResponse(fhandler *flowHandler, context *faasflow.Context, result []byte) ([]byte, error) {
	var nextNodes []*sdk.Node

	// get pipeline
	pipeline := fhandler.getPipeline()

	// Check if pipeline is active
	if pipeline.PipelineType == sdk.TYPE_DAG && !isActive(fhandler) {
		return []byte(""), fmt.Errorf("flow has been terminated")
	}

	currentNode, currentDag := pipeline.GetCurrentNodeDag()

	// Check if its a dynamic dag - handle condition/foreach subdag requests
	if currentNode.Dynamic() {
		return handleDynamicNode(fhandler, context, result)
	}

	currentNodeUniqueId := currentNode.GetUniqueId()

	// Check if the pipeline has completed excution return
	// else change depth and continue executing
	for true {
		// Get the next nodes
		nextNodes = currentNode.Children()
		// In case current node is a dynamic node stop
		if currentNode.Dynamic() {
			break
		}
		// If no nodes left
		if nextNodes == nil {
			// If depth 0 then pipeline has finished
			if pipeline.ExecutionDepth == 0 {
				fhandler.finished = true
				// TODO: Return empty for multinode
				return result, nil
			} else {

				// Update position to upper depth
				currentNode = currentDag.GetParentNode()
				currentDag = currentNode.ParentDag()
				pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_DECREMENT, currentNode.Id)
				continue
			}
		}
		break
	}
	nodeUniqueId := currentNode.GetUniqueId()

	for _, node := range nextNodes {

		var intermediateData []byte

		inDegree := node.Indegree()
		if currentNode.Dynamic() {
			// For dynamic subgraph end add the count of dynamic dependency as indegree
			// For the next node
			inDegree = inDegree + pipeline.DynamicDependencyCount[nodeUniqueId]
		}

		// Get forwarder for child node
		forwarder := currentNode.GetForwarder(node.Id)
		if forwarder != nil {
			intermediateData = forwarder(result)
		} else {
			// in case NoneForward forwarder in nil
			intermediateData = []byte{}
		}

		// If intermediate storage is enabled store data to intermediate storage
		if forwarder != nil && useIntermediateStorage() {

			key := ""
			dagNode := currentDag.GetParentNode()
			if dagNode != nil && dagNode.Dynamic() {
				// Forward data from internal node of a dynamic graph
				// <option>-<currentnodeid>-<childnodeid>
				option := pipeline.CurrentDynamicOption[dagNode.GetUniqueId()]
				key = fmt.Sprintf("%s-%s-%s", option, nodeUniqueId, node.GetUniqueId())
			} else if currentNode.Dynamic() {
				// Forward data from end node of a dynamic graph
				option := pipeline.CurrentDynamicOption[nodeUniqueId]
				key = fmt.Sprintf("%s-%s-%s", option, nodeUniqueId, node.GetUniqueId())
				delete(pipeline.CurrentDynamicOption, nodeUniqueId)
			} else {
				// <currentnodeid>-<childnodeid>
				key = fmt.Sprintf("%s-%s", nodeUniqueId, node.GetUniqueId())
			}

			serr := context.Set(key, intermediateData)
			if serr != nil {
				return []byte(""), fmt.Errorf("failed to store intermediate result, error %v", serr)
			}
			log.Printf("[Request `%s`] Intermidiate result from Node %s to %s stored as %s",
				fhandler.id, nodeUniqueId, node.GetUniqueId(), key)

			// intermediateData is set to blank once its stored in storage
			intermediateData = []byte("")
		}

		// if indegree is > 1 then use statestore to get request state
		if inDegree > 1 {
			//key := fmt.Sprintf("%s-%s", currentDag.Id, node.Id)
			key := node.GetUniqueId()
			// Update the state of indegree completion and get the updated state
			inDegreeUpdatedCount, err := fhandler.stateStore.IncrementCounter(key)
			if err != nil {
				return []byte(""), fmt.Errorf("failed to update inDegree counter for node %s", node.GetUniqueId())
			}

			// If all indegree has finished call that node
			if inDegree > inDegreeUpdatedCount {
				log.Printf("[Request `%s`] request for Node %s is delayed, completed indegree: %d/%d",
					fhandler.id, node.GetUniqueId(), inDegreeUpdatedCount, inDegree)
				continue
			}
		}

		// Set the DagExecutionPosition as of next Node
		pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_SAME, node.Id)

		// forward the flow request
		resp, forwardErr := forwardAsync(fhandler, currentNodeUniqueId, intermediateData)
		if forwardErr != nil {
			// reset dag execution position
			pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_SAME, currentNode.Id)
			return nil, fmt.Errorf("Node(%s): error: %v, %s, url %s",
				node.GetUniqueId(), forwardErr,
				string(resp), fhandler.asyncUrl)
		}

		log.Printf("[Request `%s`] Async request submitted for Node %s",
			fhandler.id, node.GetUniqueId())

	}

	// reset dag execution position
	pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_SAME, currentNode.Id)

	return []byte(""), nil
}

// isActive Checks if pipeline is active
func isActive(fhandler *flowHandler) bool {

	state, err := fhandler.stateStore.GetState()
	if err != nil {
		log.Printf("[Request `%s`] Failed to obtain pipeline state", fhandler.id)
		return false
	}

	return state
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

// getDagIntermediateData gets the intermediate data from earlier vertex
func getDagIntermediateData(handler *flowHandler, context *faasflow.Context, data []byte) ([]byte, error) {
	pipeline := handler.getPipeline()
	currentNode, dag := pipeline.GetCurrentNodeDag()
	dataMap := make(map[string][]byte)

	dependencyCount := 0

	dependencies := currentNode.Dependency()

	// XXX - handle data from parent Node - and multiple options of prev node
	for _, node := range dependencies {

		dependencyCount = dependencyCount + 1

		if node.Dynamic() {
			subDataMap := make(map[string][]byte)
			// Recive data from end node of a dynamic graph
			for _, option := range pipeline.AllDynamicOption[node.GetUniqueId()] {
				// <option>
				key := fmt.Sprintf("%s", option)
				idata, gerr := context.GetBytes(key)
				if gerr != nil {
					gerr := fmt.Errorf("key %s, %v", key, gerr)
					return data, gerr
				}
				log.Printf("[Request `%s`] Intermidiate result from Node %s to Node %s for option %s retrived from %s",
					handler.id, node.GetUniqueId(), node.GetUniqueId(),
					option, key)
				context.Del(key)

				subDataMap[option] = idata
			}

			aggregator := node.GetSubAggregator()
			idata, serr := aggregator(dataMap)
			if serr != nil {
				serr := fmt.Errorf("failed to aggregate dynamic node data, error %v", serr)
				return data, serr
			}
			delete(pipeline.AllDynamicOption, node.GetUniqueId())

			// Skip storage if NoDataForward is specified
			if node.GetForwarder(currentNode.Id) == nil {
				continue
			}

			dataMap[node.Id] = idata
		} else {

			// Skip if NoDataForward is specified
			if node.GetForwarder(currentNode.Id) == nil {
				continue
			}

			key := ""
			dagNode := dag.GetParentNode()
			if dagNode != nil && dagNode.Dynamic() {
				// Recive data from internal node of a dynamic graph
				// <option>-<dependencynodeid>-<currentnodeid>
				option := pipeline.CurrentDynamicOption[dagNode.GetUniqueId()]
				key = fmt.Sprintf("%s-%s-%s", option, node.GetUniqueId(), currentNode.GetUniqueId())
			} else {
				// <dependencynodeid>-<currentnodeid>
				key = fmt.Sprintf("%s-%s", node.GetUniqueId(), currentNode.GetUniqueId())
			}
			idata, gerr := context.GetBytes(key)
			if gerr != nil {
				gerr := fmt.Errorf("key %s, %v", key, gerr)
				return data, gerr
			}
			log.Printf("[Request `%s`] Intermidiate result from Node %s to Node %s retrived from %s",
				handler.id, node.GetUniqueId(), node.GetUniqueId(), key)
			context.Del(key)

			dataMap[node.Id] = idata
		}
	}

	// Avail the non aggregated input at context
	context.NodeInput = dataMap

	// If it has only one indegree assign the result as a data
	if len(dependencies) == 1 {
		data = dataMap[dependencies[0].Id]
	}

	aggregator := currentNode.GetAggregator()
	if aggregator != nil {
		sdata, serr := aggregator(dataMap)
		if serr != nil {
			serr := fmt.Errorf("failed to aggregate data, error %v", serr)
			return data, serr
		}
		data = sdata
	}

	return data, nil
}

// initializeStore initialize the store and return true if both store if overriden
func initializeStore(fhandler *flowHandler) (stateSDefined bool, dataSOverride bool, err error) {

	stateSDefined = false
	dataSOverride = false

	// Initialize the stateS
	stateS, err := function.DefineStateStore()
	if err != nil {
		return
	}
	if stateS != nil {
		fhandler.stateStore = stateS
		stateSDefined = true
		fhandler.stateStore.Configure(flowName, fhandler.id)
		// If request is not partial initialize the stateStore
		if !fhandler.partial {
			err = fhandler.stateStore.Init()
			if err != nil {
				return
			}
		}
	}

	// Initialize the dataS
	dataS, err := function.DefineDataStore()
	if err != nil {
		return
	}
	if dataS != nil {
		fhandler.dataStore = dataS
		dataSOverride = true
	}
	fhandler.dataStore.Configure(flowName, fhandler.id)
	// If request is not partial initialize the dataStore
	if !fhandler.partial {
		err = fhandler.dataStore.Init()
	}

	return
}

// handleWorkflow handle the flow
func handleWorkflow(data []byte) string {

	var resp []byte
	var gerr error

	// Get flow name
	flowName = getWorkflowName()
	if flowName == "" {
		log.Fatalf("Error: workflow_name must be provided, specify workflow_name: <fucntion_name> using environment")
	}

	// initialize traceserve if tracing enabled
	err := initGlobalTracer(flowName)
	if err != nil {
		log.Printf(err.Error())
	}

	// Pipeline Execution Steps
	{
		// BUILD: build the flow based on execution request
		fhandler, data := buildWorkflow(data)

		// INIT STORE: Get definition of StateStore and DataStore
		stateSDefined, dataSOverride, err := initializeStore(fhandler)
		if err != nil {
			log.Fatalf("[Request `%s`] Failed to init flow, %v", fhandler.id, err)
		}

		// Check if the pipeline is active
		if fhandler.getPipeline().PipelineType == sdk.TYPE_DAG && fhandler.partial && !isActive(fhandler) {
			log.Fatalf("flow has been terminated")
		}

		// MAKE CONTEXT: make the request context from flow
		context := createContext(fhandler)

		// DEFINE: Get Pipeline definition from user implemented Define()
		err = function.Define(fhandler.flow, context)
		if err != nil {
			log.Fatalf("[Request `%s`] Failed to define flow, %v", fhandler.id, err)
		}

		// VALIDATE: Validate Pipeline Definition
		// Dag need to be valid
		err = fhandler.getPipeline().Dag.Validate()
		if err != nil {
			log.Fatalf("[Request `%s`] Invalid dag, %v", fhandler.id, err)
		}

		// For DAG, DataStore and StateStore need to be external
		if fhandler.getPipeline().PipelineType == sdk.TYPE_DAG {
			if !stateSDefined {
				log.Fatalf("[Request `%s`] Failed, DAG flow need external StateStore", fhandler.id)
			}
			if !fhandler.getPipeline().Dag.IsExecutionFlow() && !dataSOverride {
				log.Fatalf("[Request `%s`] Failed not an execution flow, DAG data flow need external DataStore", fhandler.id)
			}
		}

		// For a new dag pipeline Create the vertex in stateStore
		if fhandler.getPipeline().PipelineType == sdk.TYPE_DAG && !fhandler.partial {
			err = fhandler.stateStore.Create(fhandler.getPipeline().GetAllNodesUniqueId())
			if err != nil {
				log.Fatalf("[Request `%s`] DAG state can not be initiated at StateStore, %v", fhandler.id, err)
			}
			serr := fhandler.stateStore.SetState(true)
			if serr != nil {
				log.Fatalf("[Request `%s`] Failed to mark dag state, error %v", fhandler.id, serr)
			}
		}

		// If not a partial request set the execution position to initial node
		if !fhandler.partial {
			// On the 0th depth set the initial node as the current execution position
			fhandler.getPipeline().UpdatePipelineExecutionPosition(sdk.DEPTH_SAME, fhandler.getPipeline().GetInitialNodeId())
		}

		// GETDATA: Get intermediate data from data store if not using default
		// For partially completed requests if IntermidiateStorage is enabled
		// get the data from dataStoretore
		if fhandler.partial && useIntermediateStorage() {

			data, gerr = getDagIntermediateData(fhandler, context, data)

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

// handleDotGraph() handle the dot graph request
func handleDotGraph() string { // Get flow name
	flowName = getWorkflowName()

	fhandler := newWorkflowHandler("", flowName, "", "", nil)
	context := createContext(fhandler)

	err := function.Define(fhandler.flow, context)
	if err != nil {
		log.Fatalf("failed to generate dot graph, error %v", err)
	}

	// VALIDATE: Validate Pipeline Definition
	// Dag need to be valid
	err = fhandler.getPipeline().Dag.Validate()
	if err != nil {
		log.Fatalf("[Request `%s`] Invalid dag, %v", fhandler.id, err)
	}

	return fhandler.getPipeline().MakeDotGraph()
}

// isDotGraphRequest check if the request is for dot graph generation
func isDotGraphRequest() bool {
	values, err := url.ParseQuery(os.Getenv("Http_Query"))
	if err != nil {
		return false
	}
	if strings.ToUpper(values.Get("generate-dot-graph")) == "TRUE" {
		return true
	}
	return false
}

// handle handle the requests
func handle(data []byte) string {
	if isDotGraphRequest() {
		return handleDotGraph()
	}
	return handleWorkflow(data)
}
