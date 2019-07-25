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
	"strconv"
	"strings"

	hmac "github.com/alexellis/hmac"
	xid "github.com/rs/xid"
	faasflow "github.com/s8sg/faas-flow"
	sdk "github.com/s8sg/faas-flow/sdk"

	gosdk "github.com/openfaas-incubator/go-function-sdk"
)

const (
	// A signature of SHA265 equivalent of "github.com/s8sg/faas-flow"
	defaultHmacKey          = "71F1D3011F8E6160813B4997BA29856744375A7F26D427D491E1CCABD4627E7C"
	counterUpdateRetryCount = 10
)

var (
	// flowName the name of the flow
	flowName = ""
)

type flowHandler struct {
	flow *faasflow.Workflow // the faasflow

	// the pipeline properties
	hasBranch       bool // State if the pipeline dag has atleast one branch
	hasEdge         bool // State if pipeline dag has atleast one edge
	isExecutionFlow bool // State if pipeline has execution only branches

	id          string // the unique request id
	query       string // the query string by user
	asyncUrl    string // the async URL of the flow
	pipelineDef []byte // the pipeline definition
	partial     bool   // denotes the flow is in partial execution state
	finished    bool   // denots the flow has finished execution

	tracer *traceHandler // Handler tracing

	stateStore faasflow.StateStore // the state store
	dataStore  faasflow.DataStore  // the data store
}

func (fhandler *flowHandler) SetRequestState(state bool) error {
	return fhandler.stateStore.Set("request-state", "true")
}

func (fhandler *flowHandler) GetRequestState() (bool, error) {
	value, err := fhandler.stateStore.Get("request-state")
	if value == "true" {
		return true, nil
	}
	return false, err
}

func (fhandler *flowHandler) SetDynamicBranchOptions(nodeUniqueId string, options []string) error {
	encoded, err := json.Marshal(options)
	if err != nil {
		return err
	}
	return fhandler.stateStore.Set(nodeUniqueId, string(encoded))
}

func (fhandler *flowHandler) GetDynamicBranchOptions(nodeUniqueId string) ([]string, error) {
	encoded, err := fhandler.stateStore.Get(nodeUniqueId)
	if err != nil {
		return nil, err
	}
	var option []string
	err = json.Unmarshal([]byte(encoded), &option)
	return option, err
}

// IncrementCounter increment counter by given term, if doesn't exist init with incrementby
func (fhandler *flowHandler) IncrementCounter(counter string, incrementby int) (int, error) {
	var serr error
	count := 0
	for i := 0; i < counterUpdateRetryCount; i++ {
		encoded, err := fhandler.stateStore.Get(counter)
		if err != nil {
			// if doesn't exist try to create
			err := fhandler.stateStore.Set(counter, fmt.Sprintf("%d", incrementby))
			if err != nil {
				return 0, fmt.Errorf("failed to update counter %s, error %v", counter, err)
			}
			return incrementby, nil
		}

		current, err := strconv.Atoi(encoded)
		if err != nil {
			return 0, fmt.Errorf("failed to update counter %s, error %v", counter, err)
		}

		count = current + incrementby
		counterStr := fmt.Sprintf("%d", count)

		err = fhandler.stateStore.Update(counter, encoded, counterStr)
		if err == nil {
			return count, nil
		}
		serr = err
	}
	return 0, fmt.Errorf("failed to update counter after max retry for %s, error %v", counter, serr)
}

func (fhandler *flowHandler) RetriveCounter(counter string) (int, error) {
	encoded, err := fhandler.stateStore.Get(counter)
	if err != nil {
		return 0, fmt.Errorf("failed to get counter %s, error %v", counter, err)
	}
	current, err := strconv.Atoi(encoded)
	if err != nil {
		return 0, fmt.Errorf("failed to get counter %s, error %v", counter, err)
	}
	return current, nil
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
		key = defaultHmacKey
	}
	return key
}

// getWorkflowName returns the flow name from env
func getWorkflowName() string {
	return os.Getenv("workflow_name")
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

// createContext create a context from request handler
func createContext(fhandler *flowHandler) *faasflow.Context {
	context := faasflow.CreateContext(fhandler.id, "",
		flowName, fhandler.dataStore)
	context.Query, _ = url.ParseQuery(fhandler.query)

	return context
}

// buildWorkflow builds a flow and context from raw request or partial completed request
func buildWorkflow(data []byte, queryString string,
	header http.Header,
	tracer *traceHandler) (fhandler *flowHandler, requestData []byte) {

	requestId := ""
	var validateErr error

	if hmacEnabled() {
		digest := ""
		digest = header.Get("X-Hub-Signature")
		key := getHmacKey()
		if len(digest) > 0 {
			validateErr = hmac.Validate(data, digest, key)
		} else {
			validateErr = fmt.Errorf("X-Hub-Signature is not set")
		}
	}

	// decode the request to find if flow definition exists
	request, err := decodeRequest(data)
	switch {
	case err != nil:
		// New Request
		if verifyRequest() {
			if validateErr != nil {
				panic(fmt.Sprintf("Failed to verify incoming request with Hmac, %v",
					validateErr.Error()))
			} else {
				fmt.Printf("Incoming request verified successfully")
			}

		}

		// Generate request Id
		requestId = xid.New().String()
		fmt.Printf("[Request `%s`] Created\n", requestId)

		// create flow properties
		dataStore := createDataStore()

		// Create fhandler
		fhandler = newWorkflowHandler("http://"+getGateway(), flowName,
			requestId, queryString, dataStore)

		// set request data
		requestData = data

		// Mark partial request
		fhandler.partial = false

		// trace req - mark as start of req
		tracer.startReqSpan(requestId)
	default:
		// Partial Request
		// Get the request ID
		requestId = request.getID()
		fmt.Printf("[Request `%s`] Received\n", requestId)

		if hmacEnabled() {
			if validateErr != nil {
				panic(fmt.Sprintf("[Request `%s`] Invalid Hmac, %v",
					requestId, validateErr.Error()))
			} else {
				fmt.Printf("[Request `%s`] Valid Hmac\n", requestId)
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
		tracer.continueReqSpan(requestId, header)
	}
	fhandler.tracer = tracer
	return
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

	var method string
	if method, ok := headers["method"]; !ok {
		method = os.Getenv("default-method")
		if method == "" {
			method = "POST"
		}
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

	var method string
	if method, ok := headers["method"]; !ok {
		method = os.Getenv("default-method")
		if method == "" {
			method = "POST"
		}
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

// findCurrentNodeToExecute find right node to execute based on state
func findCurrentNodeToExecute(fhandler *flowHandler) {
	currentNode, currentDag := fhandler.getPipeline().GetCurrentNodeDag()

	fmt.Printf("[Request `%s`] Executing node %s\n", fhandler.id, currentNode.GetUniqueId())

	// recurse to the subdag - if a node is dynamic stop to evaluate it
	for true {
		// break if request is dynamic
		if currentNode.Dynamic() {
			return
		}
		subdag := currentNode.SubDag()
		if subdag == nil {
			break
		}
		// trace node - mark as start of the parent node
		fhandler.tracer.startNodeSpan(currentNode.GetUniqueId(), fhandler.id)
		currentDag = subdag
		currentNode = currentDag.GetInitialNode()
		fmt.Printf("[Request `%s`] Executing node %s\n", fhandler.id, currentNode.GetUniqueId())
		fhandler.getPipeline().UpdatePipelineExecutionPosition(sdk.DEPTH_INCREMENT, currentNode.Id)
	}
}

// execute executes a node on a faas-flow dag
func execute(fhandler *flowHandler, request []byte) ([]byte, error) {
	var result []byte
	var err error

	pipeline := fhandler.getPipeline()

	currentNode, _ := pipeline.GetCurrentNodeDag()

	// trace node - mark as start of node
	fhandler.tracer.startNodeSpan(currentNode.GetUniqueId(), fhandler.id)

	// Execute all operation
	for _, operation := range currentNode.Operations() {

		switch {
		// If function
		case operation.Function != "":
			fmt.Printf("[Request `%s`] Executing function `%s`\n",
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
					return nil, err
				}
			}
		// If callback
		case operation.CallbackUrl != "":
			fmt.Printf("[Request `%s`] Executing callback `%s`\n",
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
					return nil, err
				}
			}

		// If modifier
		default:
			fmt.Printf("[Request `%s`] Executing modifier\n", fhandler.id)
			if result == nil {
				result, err = operation.Mod(request)
			} else {
				result, err = operation.Mod(result)
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
	}

	fmt.Printf("[Request `%s`] Completed execution of Node %s\n", fhandler.id, currentNode.GetUniqueId())

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
		return resdata, fmt.Errorf(res.Status)
	}
	return resdata, nil
}

func executeDynamic(fhandler *flowHandler, context *faasflow.Context, result []byte) ([]byte, error) {
	// get pipeline
	pipeline := fhandler.getPipeline()

	currentNode, _ := pipeline.GetCurrentNodeDag()

	// trace node - mark as start of the dynamic node
	fhandler.tracer.startNodeSpan(currentNode.GetUniqueId(),
		fhandler.id)

	currentNodeUniqueId := currentNode.GetUniqueId()
	defer fhandler.tracer.stopNodeSpan(currentNodeUniqueId)

	fmt.Printf("[Request `%s`] Processing dynamic node %s\n", fhandler.id, currentNodeUniqueId)

	// subresults and subdags
	subresults := make(map[string][]byte)
	subdags := make(map[string]*sdk.Dag)
	options := []string{}

	condition := currentNode.GetCondition()
	foreach := currentNode.GetForEach()

	switch {
	case condition != nil:
		fmt.Printf("[Request `%s`] Executing condition\n", fhandler.id)
		conditions := condition(result)
		if conditions == nil {
			panic(fmt.Sprintf("Condition function at %s returned nil, failed to proceed",
				currentNodeUniqueId))
		}
		for _, conditionKey := range conditions {
			if len(conditionKey) == 0 {
				panic(fmt.Sprintf("Condition function at %s returned invalid condiiton, failed to proceed",
					currentNodeUniqueId))
			}
			subdags[conditionKey] = currentNode.GetConditionalDag(conditionKey)
			if subdags[conditionKey] == nil {
				panic(fmt.Sprintf("Condition function at %s returned invalid condiiton, failed to proceed",
					currentNodeUniqueId))
			}
			subresults[conditionKey] = result
			options = append(options, conditionKey)
		}
	case foreach != nil:
		fmt.Printf("[Request `%s`] Executing foreach\n", fhandler.id)
		foreachResults := foreach(result)
		if foreachResults == nil {
			panic(fmt.Sprintf("Foreach function at %s returned nil, failed to proceed",
				currentNodeUniqueId))
		}
		for foreachKey, foreachResult := range foreachResults {
			if len(foreachKey) == 0 {
				panic(fmt.Sprintf("Foreach function at %s returned invalid key, failed to proceed",
					currentNodeUniqueId))
			}
			subdags[foreachKey] = currentNode.SubDag()
			subresults[foreachKey] = foreachResult
			options = append(options, foreachKey)
		}
	}

	branchCount := len(options)
	if branchCount == 0 {
		return nil, fmt.Errorf("[Request `%s`] Dynamic Node %s, failed to execute as condition/foreach returned no option",
			fhandler.id, currentNodeUniqueId)
	}

	// Set the no of branch completion for the current dynamic node
	key := pipeline.GetNodeExecutionUniqueId(currentNode) + "-branch-completion"
	_, err := fhandler.IncrementCounter(key, 0)
	if err != nil {
		return nil, fmt.Errorf("[Request `%s`] Failed to initiate dynamic indegree count for %s, err %v",
			fhandler.id, key, err)
	}

	fmt.Printf("[Request `%s`] Dynamic indegree count initiated as %s\n",
		fhandler.id, key)

	// Set all the dynamic options for the current dynamic node
	key = pipeline.GetNodeExecutionUniqueId(currentNode) + "-dynamic-branch-options"
	err = fhandler.SetDynamicBranchOptions(key, options)
	if err != nil {
		return nil, fmt.Errorf("[Request `%s`] Dynamic Node %s, failed to store dynamic options",
			fhandler.id, currentNodeUniqueId)
	}

	fmt.Printf("[Request `%s`] Dynamic options initiated as %s\n",
		fhandler.id, key)

	for option, subdag := range subdags {

		subNode := subdag.GetInitialNode()
		intermediateData := subresults[option]

		// If forwarder is not nil its not an execution flow
		if currentNode.GetForwarder("dynamic") != nil {
			key := fmt.Sprintf("%s--%s--%s", option,
				pipeline.GetNodeExecutionUniqueId(currentNode), subNode.GetUniqueId())

			serr := context.Set(key, intermediateData)
			if serr != nil {
				return []byte(""), fmt.Errorf("failed to store intermediate result, error %v", serr)
			}
			fmt.Printf("[Request `%s`] Intermidiate result for option %s from Node %s to %s stored as %s\n",
				fhandler.id, option, currentNodeUniqueId, subNode.GetUniqueId(), key)

			// intermediateData is set to blank once its stored in storage
			intermediateData = []byte("")
		}

		// Increment the depth to execute the dynamic branch
		pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_INCREMENT, subNode.Id)
		// Set the option the dynamic branch is performing
		pipeline.CurrentDynamicOption[currentNode.GetUniqueId()] = option

		// forward the flow request
		resp, forwardErr := forwardAsync(fhandler,
			currentNode.GetUniqueId(), intermediateData)
		if forwardErr != nil {
			// reset dag execution position
			pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_DECREMENT, currentNode.Id)
			return nil, fmt.Errorf("Node(%s): error: %v, %s, url %s",
				currentNode.GetUniqueId(), forwardErr,
				string(resp), fhandler.asyncUrl)
		}

		fmt.Printf("[Request `%s`] Async request submitted for Node %s option %s\n",
			fhandler.id, subNode.GetUniqueId(), option)

		// reset pipeline
		pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_DECREMENT, currentNode.Id)
		delete(pipeline.CurrentDynamicOption, currentNode.GetUniqueId())
	}

	return []byte(""), nil
}

// findNextNodeToExecute find the next node(s) to execute after the current node
func findNextNodeToExecute(fhandler *flowHandler) bool {
	// get pipeline
	pipeline := fhandler.getPipeline()

	// Check if pipeline is active in statestore when executing dag with branches
	if fhandler.hasBranch && !isActive(fhandler) {
		fmt.Printf("[Request `%s`] Pipeline has been terminated\n", fhandler.id)
		panic(fmt.Sprintf("[Request `%s`] Pipeline has been terminated", fhandler.id))
	}

	currentNode, currentDag := pipeline.GetCurrentNodeDag()
	// Check if the pipeline has completed excution return
	// else change depth and continue executing
	for true {
		defer fhandler.tracer.stopNodeSpan(currentNode.GetUniqueId())

		// If nodes left in current dag return
		if currentNode.Children() != nil {
			return true
		}

		// If depth 0 then pipeline has finished
		if pipeline.ExecutionDepth == 0 {
			fhandler.finished = true
			return false
		} else {
			// Update position to lower depth
			currentNode = currentDag.GetParentNode()
			currentDag = currentNode.ParentDag()
			pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_DECREMENT, currentNode.Id)
			fmt.Printf("[Request `%s`] Executing Node %s", fhandler.id, currentNode.GetUniqueId())

			// mark execution of the node for new depth
			fhandler.tracer.startNodeSpan(currentNode.GetUniqueId(), fhandler.id)

			// If current node is a dynamic node, forward the request for its end
			if currentNode.Dynamic() {
				break
			}
		}
	}
	return false
}

// handleDynamicEnd
func handleDynamicEnd(fhandler *flowHandler, context *faasflow.Context, result []byte) ([]byte, error) {

	pipeline := fhandler.getPipeline()
	currentNode, _ := pipeline.GetCurrentNodeDag()

	// Get dynamic options computed for the current dynamic node
	key := pipeline.GetNodeExecutionUniqueId(currentNode) + "-dynamic-branch-options"
	options, err := fhandler.GetDynamicBranchOptions(key)
	if err != nil {
		return nil, fmt.Errorf("failed to retrive dynamic options for %v, error %v",
			currentNode.GetUniqueId(), err)
	}
	// Get unique execution id of the node
	branchkey := pipeline.GetNodeExecutionUniqueId(currentNode) + "-branch-completion"

	// if indegree is > 1 then use statestore to get indegree completion state
	if len(options) > 1 {

		// Get unique execution id of the node
		key = pipeline.GetNodeExecutionUniqueId(currentNode) + "-branch-completion"
		// Update the state of indegree completion and get the updated state

		// Skip if dynamic node data forwarding is not disabled
		if currentNode.GetForwarder("dynamic") != nil {
			option := pipeline.CurrentDynamicOption[currentNode.GetUniqueId()]
			key = fmt.Sprintf("%s--%s--%s", option, pipeline.GetNodeExecutionUniqueId(currentNode), currentNode.GetUniqueId())

			err = context.Set(key, result)
			if err != nil {
				return nil, fmt.Errorf("failed to store branch result of dynamic node %s for option %s, error %v",
					currentNode.GetUniqueId(), option, err)
			}
			fmt.Printf("[Request `%s`] Intermidiate result from Branch to Dynamic Node %s for option %s stored as %s\n",
				fhandler.id, currentNode.GetUniqueId(), option, key)
		}
		realIndegree, err := fhandler.IncrementCounter(branchkey, 1)
		if err != nil {
			return []byte(""), fmt.Errorf("failed to update inDegree counter for node %s", currentNode.GetUniqueId())
		}

		fmt.Printf("[Request `%s`] Executing end of dynamic node %s, completed indegree: %d/%d\n",
			fhandler.id, currentNode.GetUniqueId(), realIndegree, len(options))

		//not last branch return
		if realIndegree < len(options) {
			return nil, nil
		}
	} else {
		fmt.Printf("[Request `%s`] Executing end of dynamic node %s, branch count 1\n",
			fhandler.id, currentNode.GetUniqueId())
	}
	// get current execution option
	currentOption := pipeline.CurrentDynamicOption[currentNode.GetUniqueId()]

	// skip aggregating if Data Forwarder is disabled for dynamic node
	if currentNode.GetForwarder("dynamic") == nil {
		return []byte(""), nil
	}

	subDataMap := make(map[string][]byte)
	// Store the current data
	subDataMap[currentOption] = result

	// Recive data from a dynamic graph for each options
	for _, option := range options {
		key := fmt.Sprintf("%s--%s--%s",
			option, pipeline.GetNodeExecutionUniqueId(currentNode), currentNode.GetUniqueId())

		// skip retriving data for current option
		if option == currentOption {
			context.Del(key)
			continue
		}

		idata := context.GetBytes(key)
		fmt.Printf("[Request `%s`] Intermidiate result from Branch to Dynamic Node %s for option %s retrived from %s\n",
			fhandler.id, currentNode.GetUniqueId(), option, key)
		// delete intermidiate data after retrival
		context.Del(key)

		subDataMap[option] = idata
	}

	// Get SubAggregator and call
	fmt.Printf("[Request `%s`] Executing aggregator of Dynamic Node %s\n",
		fhandler.id, currentNode.GetUniqueId())
	aggregator := currentNode.GetSubAggregator()
	data, serr := aggregator(subDataMap)
	if serr != nil {
		serr := fmt.Errorf("failed to aggregate dynamic node data, error %v", serr)
		return nil, serr
	}
	if data == nil {
		data = []byte("")
	}

	return data, nil
}

// handleNextNodes Handle request Response for a faasflow perform response/asyncforward
func handleNextNodes(fhandler *flowHandler, context *faasflow.Context, result []byte) ([]byte, error) {
	// get pipeline
	pipeline := fhandler.getPipeline()

	currentNode, _ := pipeline.GetCurrentNodeDag()
	nextNodes := currentNode.Children()

	for _, node := range nextNodes {

		var intermediateData []byte

		// Node's total Indegree
		inDegree := node.Indegree()

		// Get forwarder for child node
		forwarder := currentNode.GetForwarder(node.Id)

		if forwarder != nil {
			// call default or user defined forwarder
			intermediateData = forwarder(result)
			key := fmt.Sprintf("%s--%s", pipeline.GetNodeExecutionUniqueId(currentNode),
				node.GetUniqueId())

			serr := context.Set(key, intermediateData)
			if serr != nil {
				return []byte(""), fmt.Errorf("failed to store intermediate result, error %v", serr)
			}
			fmt.Printf("[Request `%s`] Intermidiate result from Node %s to %s stored as %s\n",
				fhandler.id, currentNode.GetUniqueId(), node.GetUniqueId(), key)

			// intermediateData is set to blank once its stored in storage
			intermediateData = []byte("")
		} else {
			// in case NoneForward forwarder in nil
			intermediateData = []byte("")
		}

		// if indegree is > 1 then use statestore to get indegree completion state
		if inDegree > 1 {
			// Update the state of indegree completion and get the updated state
			key := pipeline.GetNodeExecutionUniqueId(node)
			inDegreeUpdatedCount, err := fhandler.IncrementCounter(key, 1)
			if err != nil {
				return []byte(""), fmt.Errorf("failed to update inDegree counter for node %s", node.GetUniqueId())
			}

			// If all indegree has finished call that node
			if inDegree > inDegreeUpdatedCount {
				fmt.Printf("[Request `%s`] request for Node %s is delayed, completed indegree: %d/%d\n",
					fhandler.id, node.GetUniqueId(), inDegreeUpdatedCount, inDegree)
				continue
			} else {
				fmt.Printf("[Request `%s`] performing request for Node %s, completed indegree: %d/%d\n",
					fhandler.id, node.GetUniqueId(), inDegreeUpdatedCount, inDegree)
			}
		} else {
			fmt.Printf("[Request `%s`] performing request for Node %s, indegree count is 1\n",
				fhandler.id, node.GetUniqueId())
		}

		// Set the DagExecutionPosition as of next Node
		pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_SAME, node.Id)

		// forward the flow request
		resp, forwardErr := forwardAsync(fhandler, currentNode.GetUniqueId(),
			intermediateData)
		if forwardErr != nil {
			// reset dag execution position
			pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_SAME, currentNode.Id)
			return nil, fmt.Errorf("Node(%s): error: %v, %s, url %s",
				node.GetUniqueId(), forwardErr,
				string(resp), fhandler.asyncUrl)
		}

		fmt.Printf("[Request `%s`] Async request submitted for Node %s\n",
			fhandler.id, node.GetUniqueId())

	}

	// reset dag execution position
	pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_SAME, currentNode.Id)

	return []byte(""), nil
}

// isActive Checks if pipeline is active
func isActive(fhandler *flowHandler) bool {

	state, err := fhandler.GetRequestState()
	if err != nil {
		fmt.Printf("[Request `%s`] Failed to obtain pipeline state\n", fhandler.id)
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
		fmt.Printf("[Request `%s`] Calling failure handler for error, %v\n",
			fhandler.id, err)
		data, err = fhandler.getPipeline().FailureHandler(err)
	}

	fhandler.finished = true

	// call finally handler if available
	if fhandler.getPipeline().Finally != nil {
		fmt.Printf("[Request `%s`] Calling Finally handler with state: %s\n",
			fhandler.id, faasflow.StateFailure)
		fhandler.getPipeline().Finally(faasflow.StateFailure)
	}
	if data != nil {
		fmt.Printf("%s", string(data))
	}

	// Cleanup data and state for failure
	if fhandler.stateStore != nil {
		fhandler.stateStore.Cleanup()
	}
	fhandler.dataStore.Cleanup()

	// stop req span if request has finished
	fhandler.tracer.stopReqSpan()

	// flash any pending trace item if tracing enabled
	fhandler.tracer.flushTracer()

	panic(fmt.Sprintf("[Request `%s`] Failed, %v\n", fhandler.id, err))
}

// getDagIntermediateData gets the intermediate data from earlier vertex
func getDagIntermediateData(handler *flowHandler, context *faasflow.Context) ([]byte, error) {
	var data []byte

	pipeline := handler.getPipeline()
	currentNode, dag := pipeline.GetCurrentNodeDag()
	dataMap := make(map[string][]byte)

	var scenarioFirstNodeOfDynamicBranch bool

	// if current node is the initial node of a dynamic branch dag
	dagNode := dag.GetParentNode()
	if dagNode != nil && dagNode.Dynamic() &&
		dag.GetInitialNode() == currentNode {
		scenarioFirstNodeOfDynamicBranch = true
	}

	switch {
	// handle if current node is the initial node of a dynamic branch dag
	case scenarioFirstNodeOfDynamicBranch:
		// Skip if NoDataForward is specified
		if dagNode.GetForwarder("dynamic") != nil {
			option := pipeline.CurrentDynamicOption[dagNode.GetUniqueId()]
			// Option not get appended in key as its already in state
			key := fmt.Sprintf("%s--%s", pipeline.GetNodeExecutionUniqueId(dagNode), currentNode.GetUniqueId())
			data = context.GetBytes(key)
			fmt.Printf("[Request `%s`] Intermidiate result from Node %s to Node %s for option %s retrived from %s\n",
				handler.id, dagNode.GetUniqueId(), currentNode.GetUniqueId(),
				option, key)
			// delete intermidiate data after retrival
			context.Del(key)
		}

	// handle normal scenario
	default:
		dependencies := currentNode.Dependency()
		// current node has dependencies in same dag
		for _, node := range dependencies {

			// Skip if NoDataForward is specified
			if node.GetForwarder(currentNode.Id) == nil {
				continue
			}

			key := fmt.Sprintf("%s--%s", pipeline.GetNodeExecutionUniqueId(node), currentNode.GetUniqueId())
			idata := context.GetBytes(key)
			fmt.Printf("[Request `%s`] Intermidiate result from Node %s to Node %s retrived from %s\n",
				handler.id, node.GetUniqueId(), currentNode.GetUniqueId(), key)
			// delete intermidiate data after retrival
			context.Del(key)

			dataMap[node.Id] = idata

		}

		// Avail the non aggregated input at context
		context.NodeInput = dataMap

		// If it has only one indegree assign the result as a data
		if len(dependencies) == 1 {
			data = dataMap[dependencies[0].Id]
		}

		// If Aggregator is available get agregator
		aggregator := currentNode.GetAggregator()
		if aggregator != nil {
			sdata, serr := aggregator(dataMap)
			if serr != nil {
				serr := fmt.Errorf("failed to aggregate data, error %v", serr)
				return data, serr
			}
			data = sdata
		}

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
func handleWorkflow(req *gosdk.Request) (string, string) {

	var resp []byte
	var gerr error
	var reqId string

	data := req.Body
	queryString := req.QueryString
	header := req.Header

	// Get flow name
	flowName = getWorkflowName()
	if flowName == "" {
		panic(fmt.Sprintf("Error: workflow_name must be provided, specify workflow_name: <fucntion_name> using environment"))
	}

	// initialize traceserve if tracing enabled
	tracer, err := initRequestTracer(flowName)
	if err != nil {
		fmt.Printf(err.Error())
	}

	// Pipeline Execution Steps
	{
		// BUILD: build the flow based on the request data
		fhandler, data := buildWorkflow(data, queryString, header, tracer)

		reqId = fhandler.id

		// INIT STORE: Get definition of StateStore and DataStore from user
		stateSDefined, dataSOverride, err := initializeStore(fhandler)
		if err != nil {
			panic(fmt.Sprintf("[Request `%s`] Failed to init flow, %v", fhandler.id, err))
		}

		// MAKE CONTEXT: make the request context from flow
		context := createContext(fhandler)

		// DEFINE: Get Pipeline definition from user implemented Define()
		err = function.Define(fhandler.flow, context)
		if err != nil {
			panic(fmt.Sprintf("[Request `%s`] Failed to define flow, %v", fhandler.id, err))
		}

		// VALIDATE: Validate Pipeline Definition
		err = fhandler.getPipeline().Dag.Validate()
		if err != nil {
			panic(fmt.Sprintf("[Request `%s`] Invalid dag, %v", fhandler.id, err))
		}
		fhandler.hasBranch = fhandler.getPipeline().Dag.HasBranch()
		fhandler.hasEdge = fhandler.getPipeline().Dag.HasEdge()
		fhandler.isExecutionFlow = fhandler.getPipeline().Dag.IsExecutionFlow()

		// For dag one of the branch can cause the pipeline to terminate
		// hence we Check if the pipeline is active
		if fhandler.hasBranch && fhandler.partial && !isActive(fhandler) {
			panic(fmt.Sprintf("[Request `%s`] flow has been terminated", fhandler.id))
		}

		// For dag which has branches
		// StateStore need to be external
		if fhandler.hasBranch && !stateSDefined {
			panic(fmt.Sprintf("[Request `%s`] Failed, DAG flow need external StateStore", fhandler.id))
		}

		// If dags has atleast one edge
		// and nodes forwards data, data store need to be external
		if fhandler.hasEdge && !fhandler.isExecutionFlow && !dataSOverride {
			panic(fmt.Sprintf("[Request `%s`] Failed not an execution flow, DAG data flow need external DataStore", fhandler.id))
		}

		// If not a partial request
		if !fhandler.partial {

			// For a new dag pipeline that has branches Create the vertex in stateStore
			if fhandler.hasBranch {
				serr := fhandler.SetRequestState(true)
				if serr != nil {
					panic(fmt.Sprintf("[Request `%s`] Failed to mark dag state, error %v", fhandler.id, serr))
				}
				fmt.Printf("[Request `%s`] DAG state initiated at StateStore\n", fhandler.id)
			}

			// set the execution position to initial node
			// On the 0th depth set the initial node as the current execution position
			fhandler.getPipeline().UpdatePipelineExecutionPosition(sdk.DEPTH_SAME,
				fhandler.getPipeline().GetInitialNodeId())
		}

		// if not an execution only dag, for partial request get intermediate data
		if fhandler.partial && !fhandler.getPipeline().Dag.IsExecutionFlow() {

			// Get intermediate data from data store
			data, gerr = getDagIntermediateData(fhandler, context)
			if gerr != nil {
				gerr := fmt.Errorf("failed to retrive intermediate result, error %v", gerr)
				handleFailure(fhandler, context, gerr)
			}
		}

		// Find the right node to execute now
		findCurrentNodeToExecute(fhandler)

		currentNode, _ := fhandler.getPipeline().GetCurrentNodeDag()

		result := []byte{}

		switch {
		// Execute the dynamic node
		case currentNode.Dynamic():
			result, err = executeDynamic(fhandler, context, data)
			if err != nil {
				handleFailure(fhandler, context, err)
			}
		// Execute the node
		default:
			result, err = execute(fhandler, data)
			if err != nil {
				handleFailure(fhandler, context, err)
			}

			// Find the right node to execute next
			findNextNodeToExecute(fhandler)

		NodeCompletionLoop:
			for !fhandler.finished {
				currentNode, _ := fhandler.getPipeline().GetCurrentNodeDag()
				switch {
				// Execute a end of dynamic node
				case currentNode.Dynamic():
					result, err = handleDynamicEnd(fhandler, context, result)
					if err != nil {
						handleFailure(fhandler, context, err)
					}
					// in case dynamic end can not be executed
					if result == nil {
						fhandler.tracer.stopNodeSpan(currentNode.GetUniqueId())
						break NodeCompletionLoop
					}
					// in case dynamic nodes end has finished execution,
					// find next node and continue
					// although if next is a children handle it
					notLast := findNextNodeToExecute(fhandler)
					if !notLast {
						continue
					}
					fallthrough
				default:
					// handle the execution iteration and update state
					result, err = handleNextNodes(fhandler, context, result)
					if err != nil {
						handleFailure(fhandler, context, err)
					}
					break NodeCompletionLoop
				}
			}
		}

		if fhandler.finished {
			fmt.Printf("[Request `%s`] Completed successfully\n", fhandler.id)
			context.State = faasflow.StateSuccess
			if fhandler.getPipeline().Finally != nil {
				fmt.Printf("[Request `%s`] Calling Finally handler with state: %s\n",
					fhandler.id, faasflow.StateSuccess)
				fhandler.getPipeline().Finally(faasflow.StateSuccess)
			}

			// Cleanup data and state for failure
			if fhandler.stateStore != nil {
				fhandler.stateStore.Cleanup()
			}
			fhandler.dataStore.Cleanup()

			resp = result
		}

		// stop req span if request has finished
		tracer.stopReqSpan()
	}

	// flash any pending trace item if tracing enabled
	tracer.flushTracer()

	return string(resp), reqId
}

// handleDagExport() handle the dag export request
func handleDagExport() string {
	flowName = getWorkflowName()

	fhandler := newWorkflowHandler("", flowName, "", "", nil)
	context := createContext(fhandler)

	err := function.Define(fhandler.flow, context)
	if err != nil {
		panic(fmt.Sprintf("[Request `dag-export`] failed to export graph, error %v", err))
	}

	return fhandler.getPipeline().GetDagDefinition()
}

// isDagExportRequest check if dag export request
func isDagExportRequest(req *gosdk.Request) bool {
	values, err := url.ParseQuery(req.QueryString)
	if err != nil {
		return false
	}

	if strings.ToUpper(values.Get("export-dag")) == "TRUE" {
		return true
	}
	return false
}

// handle handles the request
func handle(req *gosdk.Request, response *gosdk.Response) (err error) {

	var result string
	var reqId string

	status := http.StatusOK

	// set error inside the handler
	/*
		defer func() {
			if recoverLog := recover(); recoverLog != nil {
				err = fmt.Errorf("%v", recoverLog)
				response.Header.Set("X-Faas-Flow-ReqID", reqId)
			}
		}()
	*/

	if isDagExportRequest(req) {
		result = handleDagExport()
	} else {
		result, reqId = handleWorkflow(req)
		response.Header.Set("X-Faas-Flow-Reqid", reqId)
	}

	response.Body = []byte(result)
	response.StatusCode = status

	return
}
