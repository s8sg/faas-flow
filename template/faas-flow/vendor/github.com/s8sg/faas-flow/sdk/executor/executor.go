package executor

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	hmac "github.com/alexellis/hmac"
	xid "github.com/rs/xid"
	sdk "github.com/s8sg/faas-flow/sdk"
)

// RawRequest a raw request for the flow
type RawRequest struct {
	Data          []byte
	AuthSignature string
	Query         string
}

// ExecutionRuntime implements how operation executed and handle next nodes in async
type ExecutionRuntime interface {
	// HandleNextNode handles execution of next nodes based on partial state
	HandleNextNode(state []byte) (err error)
	// ExecuteOperation implements execution of an operation
	ExecuteOperation(sdk.Operation, []byte) ([]byte, error)
}

// Executor implements a faas-flow executor
type Executor interface {
	// Configure configure an executor with request id
	Configure(requestId string)
	// GetFlowName get nbame of the flow
	GetFlowName() string
	// GetFlowDefinition get definition of the faas-flow
	GetFlowDefinition(*sdk.Pipeline, *sdk.Context) error
	// ReqValidationEnabled check if request validation enabled
	ReqValidationEnabled() bool
	// GetValidationKey get request validation key
	GetValidationKey() (string, error)
	// ReqAuthEnabled check if request auth enabled
	ReqAuthEnabled() bool
	// GetReqAuthKey get the request auth key
	GetReqAuthKey() (string, error)
	// MonitoringEnabled check if request monitoring enabled
	MonitoringEnabled() bool
	// GetEventHandler get the event handler for request monitoring
	GetEventHandler() (sdk.EventHandler, error)
	// LoggingEnabled check if logging is enabled
	LoggingEnabled() bool
	// GetLogger get the logger
	GetLogger() (sdk.Logger, error)
	// GetStateStore get the state store
	GetStateStore() (sdk.StateStore, error)
	// GetDataStore get the datta store
	GetDataStore() (sdk.DataStore, error)

	ExecutionRuntime
}

// FlowExecutor faasflow executor
type FlowExecutor struct {
	flow *sdk.Pipeline // the faasflow

	// the pipeline properties
	hasBranch       bool // State if the pipeline dag has atleast one branch
	hasEdge         bool // State if pipeline dag has atleast one edge
	isExecutionFlow bool // State if pipeline has execution only branches

	flowName string // the name of the flow
	id       string // the unique request id
	query    string // the query to the flow

	eventHandler sdk.EventHandler // Handler flow events
	logger       sdk.Logger       // Handle flow logs
	stateStore   sdk.StateStore   // the state store
	dataStore    sdk.DataStore    // the data store

	partial      bool        // denotes the flow is in partial execution state
	newRequest   *RawRequest // holds the new request
	partialState []byte      // holds the partially complted state
	finished     bool        // denots the flow has finished execution

	executor Executor // executor
}

/*
type ExecutionOption struct {
}
type ExecutionOption func(*ExecutionOptions)
*/

type ExecutionStateOptions struct {
	newRequest   *RawRequest
	partialState []byte
}

type ExecutionStateOption func(*ExecutionStateOptions)

func NewRequest(request *RawRequest) ExecutionStateOption {
	return func(o *ExecutionStateOptions) {
		o.newRequest = request
	}
}

func PartialRequest(partialState []byte) ExecutionStateOption {
	return func(o *ExecutionStateOptions) {
		o.partialState = partialState
	}
}

const (
	// signature of SHA265 equivalent of "github.com/s8sg/faas-flow"
	defaultHmacKey = "71F1D3011F8E6160813B4997BA29856744375A7F26D427D491E1CCABD4627E7C"
	// max retry count to update counter
	counterUpdateRetryCount = 10
)

// log logs using logger if logging enabled
func (fexec *FlowExecutor) log(str string, a ...interface{}) {
	if fexec.executor.LoggingEnabled() {
		str := fmt.Sprintf(str, a...)
		fexec.logger.Log(str)
	}
}

// setRequestState set the request state
func (fexec *FlowExecutor) setRequestState(state bool) error {
	return fexec.stateStore.Set("request-state", "true")
}

// getRequestState get state of the request
func (fexec *FlowExecutor) getRequestState() (bool, error) {
	value, err := fexec.stateStore.Get("request-state")
	if value == "true" {
		return true, nil
	}
	return false, err
}

// setDynamicBranchOptions set dynamic options for a dynamic node
func (fexec *FlowExecutor) setDynamicBranchOptions(nodeUniqueId string, options []string) error {
	encoded, err := json.Marshal(options)
	if err != nil {
		return err
	}
	return fexec.stateStore.Set(nodeUniqueId, string(encoded))
}

// getDynamicBranchOptions get dynamic options for a dynamic node
func (fexec *FlowExecutor) getDynamicBranchOptions(nodeUniqueId string) ([]string, error) {
	encoded, err := fexec.stateStore.Get(nodeUniqueId)
	if err != nil {
		return nil, err
	}
	var option []string
	err = json.Unmarshal([]byte(encoded), &option)
	return option, err
}

// incrementCounter increment counter by given term, if doesn't exist init with incrementby
func (fexec *FlowExecutor) incrementCounter(counter string, incrementby int) (int, error) {
	var serr error
	count := 0
	for i := 0; i < counterUpdateRetryCount; i++ {
		encoded, err := fexec.stateStore.Get(counter)
		if err != nil {
			// if doesn't exist try to create
			err := fexec.stateStore.Set(counter, fmt.Sprintf("%d", incrementby))
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

		err = fexec.stateStore.Update(counter, encoded, counterStr)
		if err == nil {
			return count, nil
		}
		serr = err
	}
	return 0, fmt.Errorf("failed to update counter after max retry for %s, error %v", counter, serr)
}

// retriveCounter retrives a counter value
func (fexec *FlowExecutor) retriveCounter(counter string) (int, error) {
	encoded, err := fexec.stateStore.Get(counter)
	if err != nil {
		return 0, fmt.Errorf("failed to get counter %s, error %v", counter, err)
	}
	current, err := strconv.Atoi(encoded)
	if err != nil {
		return 0, fmt.Errorf("failed to get counter %s, error %v", counter, err)
	}
	return current, nil
}

// isActive check if flow is active
func (fexec *FlowExecutor) isActive() bool {
	state, err := fexec.getRequestState()
	if err != nil {
		fexec.log("[Request `%s`] Failed to obtain pipeline state\n", fexec.id)
		return false
	}

	return state
}

// executeNode  executes a node on a faas-flow dag
func (fexec *FlowExecutor) executeNode(request []byte) ([]byte, error) {
	var result []byte
	var err error

	pipeline := fexec.flow

	currentNode, _ := pipeline.GetCurrentNodeDag()

	// mark as start of node
	if fexec.executor.MonitoringEnabled() {
		fexec.eventHandler.ReportNodeStart(currentNode.GetUniqueId(), fexec.id)
	}

	for _, operation := range currentNode.Operations() {
		if fexec.executor.MonitoringEnabled() {
			fexec.eventHandler.ReportOperationStart(operation.GetId(), currentNode.GetUniqueId(), fexec.id)
		}
		if result == nil {
			result, err = fexec.executor.ExecuteOperation(operation, request)
		} else {
			result, err = fexec.executor.ExecuteOperation(operation, result)
		}
		if err != nil {
			if fexec.executor.MonitoringEnabled() {
				fexec.eventHandler.ReportOperationFailure(operation.GetId(), currentNode.GetUniqueId(), fexec.id, err)
			}
			err = fmt.Errorf("Node(%s), Operation (%s), error: execution failed, %v",
				currentNode.GetUniqueId(), operation.GetId(), err)
			return nil, err
		}
		if fexec.executor.MonitoringEnabled() {
			fexec.eventHandler.ReportOperationEnd(operation.GetId(), currentNode.GetUniqueId(), fexec.id)
		}
	}

	fexec.log("[Request `%s`] Completed execution of Node %s\n", fexec.id, currentNode.GetUniqueId())

	return result, nil
}

// findCurrentNodeToExecute find right node to execute based on state
func (fexec *FlowExecutor) findCurrentNodeToExecute() {
	currentNode, currentDag := fexec.flow.GetCurrentNodeDag()

	fexec.log("[Request `%s`] Executing node %s\n", fexec.id, currentNode.GetUniqueId())

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
		if fexec.executor.MonitoringEnabled() {
			fexec.eventHandler.ReportNodeStart(currentNode.GetUniqueId(), fexec.id)
		}
		fexec.log("[Request `%s`] Executing node %s\n", fexec.id, currentNode.GetUniqueId())
		currentDag = subdag
		currentNode = currentDag.GetInitialNode()
		fexec.flow.UpdatePipelineExecutionPosition(sdk.DEPTH_INCREMENT, currentNode.Id)
	}
}

// forwardState forward async request to faasflow
func (fexec *FlowExecutor) forwardState(currentNodeId string, result []byte) error {
	var sign string
	store := make(map[string]string)

	// get pipeline
	pipeline := fexec.flow

	// Get pipeline state
	pipelineState := pipeline.GetState()

	defaultStore, ok := fexec.dataStore.(*requestEmbedDataStore)
	if ok {
		store = defaultStore.store
	}

	// Check if request validation used
	if fexec.executor.ReqValidationEnabled() {
		key, err := fexec.executor.GetValidationKey()
		if err != nil {
			return fmt.Errorf("failed to get key, error %v", err)
		}
		hash := hmac.Sign([]byte(pipelineState), []byte(key))
		sign = "sha1=" + hex.EncodeToString(hash)
	}

	// Build request
	uprequest := buildRequest(fexec.id, string(pipelineState), fexec.query, result, store, sign)

	// Make request data
	state, _ := uprequest.encode()

	if fexec.executor.MonitoringEnabled() {
		fexec.eventHandler.ReportExecutionForward(currentNodeId, fexec.id)
	}

	err := fexec.executor.HandleNextNode(state)
	if err != nil {
		return err
	}

	return nil
}

// executeDynamic executes a dynamic node
func (fexec *FlowExecutor) executeDynamic(context *sdk.Context, result []byte) ([]byte, error) {
	// get pipeline
	pipeline := fexec.flow

	currentNode, _ := pipeline.GetCurrentNodeDag()

	// trace node - mark as start of the dynamic node
	if fexec.executor.MonitoringEnabled() {
		fexec.eventHandler.ReportNodeStart(currentNode.GetUniqueId(),
			fexec.id)
	}

	currentNodeUniqueId := currentNode.GetUniqueId()
	if fexec.executor.MonitoringEnabled() {
		defer fexec.eventHandler.ReportNodeEnd(currentNodeUniqueId, fexec.id)
	}

	fexec.log("[Request `%s`] Processing dynamic node %s\n", fexec.id, currentNodeUniqueId)

	// subresults and subdags
	subresults := make(map[string][]byte)
	subdags := make(map[string]*sdk.Dag)
	options := []string{}

	condition := currentNode.GetCondition()
	foreach := currentNode.GetForEach()

	switch {
	case condition != nil:
		fexec.log("[Request `%s`] Executing condition\n", fexec.id)
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
		fexec.log("[Request `%s`] Executing foreach\n", fexec.id)
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
			fexec.id, currentNodeUniqueId)
	}

	// Set the no of branch completion for the current dynamic node
	key := pipeline.GetNodeExecutionUniqueId(currentNode) + "-branch-completion"
	_, err := fexec.incrementCounter(key, 0)
	if err != nil {
		return nil, fmt.Errorf("[Request `%s`] Failed to initiate dynamic indegree count for %s, err %v",
			fexec.id, key, err)
	}

	fexec.log("[Request `%s`] Dynamic indegree count initiated as %s\n",
		fexec.id, key)

	// Set all the dynamic options for the current dynamic node
	key = pipeline.GetNodeExecutionUniqueId(currentNode) + "-dynamic-branch-options"
	err = fexec.setDynamicBranchOptions(key, options)
	if err != nil {
		return nil, fmt.Errorf("[Request `%s`] Dynamic Node %s, failed to store dynamic options",
			fexec.id, currentNodeUniqueId)
	}

	fexec.log("[Request `%s`] Dynamic options initiated as %s\n",
		fexec.id, key)

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
			fexec.log("[Request `%s`] Intermidiate result for option %s from Node %s to %s stored as %s\n",
				fexec.id, option, currentNodeUniqueId, subNode.GetUniqueId(), key)

			// intermediateData is set to blank once its stored in storage
			intermediateData = []byte("")
		}

		// Increment the depth to execute the dynamic branch
		pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_INCREMENT, subNode.Id)
		// Set the option the dynamic branch is performing
		pipeline.CurrentDynamicOption[currentNode.GetUniqueId()] = option

		// forward the flow request
		forwardErr := fexec.forwardState(currentNode.GetUniqueId(), intermediateData)
		if forwardErr != nil {
			// reset dag execution position
			pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_DECREMENT, currentNode.Id)
			return nil, fmt.Errorf("Node(%s): error: %v",
				currentNode.GetUniqueId(), forwardErr)
		}

		fexec.log("[Request `%s`] Async request submitted for Node %s option %s\n",
			fexec.id, subNode.GetUniqueId(), option)

		// reset pipeline
		pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_DECREMENT, currentNode.Id)
		delete(pipeline.CurrentDynamicOption, currentNode.GetUniqueId())
	}

	return []byte(""), nil
}

// findNextNodeToExecute find the next node(s) to execute after the current node
func (fexec *FlowExecutor) findNextNodeToExecute() bool {
	// get pipeline
	pipeline := fexec.flow

	// Check if pipeline is active in statestore when executing dag with branches
	if fexec.hasBranch && !fexec.isActive() {
		fexec.log("[Request `%s`] Pipeline has been terminated\n", fexec.id)
		panic(fmt.Sprintf("[Request `%s`] Pipeline has been terminated", fexec.id))
	}

	currentNode, currentDag := pipeline.GetCurrentNodeDag()
	// Check if the pipeline has completed excution return
	// else change depth and continue executing
	for true {
		if fexec.executor.MonitoringEnabled() {
			defer fexec.eventHandler.ReportNodeEnd(currentNode.GetUniqueId(), fexec.id)
		}

		// If nodes left in current dag return
		if currentNode.Children() != nil {
			return true
		}

		// If depth 0 then pipeline has finished
		if pipeline.ExecutionDepth == 0 {
			fexec.finished = true
			return false
		} else {
			// Update position to lower depth
			currentNode = currentDag.GetParentNode()
			currentDag = currentNode.ParentDag()
			pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_DECREMENT, currentNode.Id)
			fexec.log("[Request `%s`] Executing Node %s", fexec.id, currentNode.GetUniqueId())

			// mark execution of the node for new depth
			if fexec.executor.MonitoringEnabled() {
				defer fexec.eventHandler.ReportNodeStart(currentNode.GetUniqueId(), fexec.id)
			}

			// If current node is a dynamic node, forward the request for its end
			if currentNode.Dynamic() {
				break
			}
		}
	}
	return false
}

// handleDynamicEnd handles the end of a dynamic node
func (fexec *FlowExecutor) handleDynamicEnd(context *sdk.Context, result []byte) ([]byte, error) {

	pipeline := fexec.flow
	currentNode, _ := pipeline.GetCurrentNodeDag()

	// Get dynamic options computed for the current dynamic node
	key := pipeline.GetNodeExecutionUniqueId(currentNode) + "-dynamic-branch-options"
	options, err := fexec.getDynamicBranchOptions(key)
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
			fexec.log("[Request `%s`] Intermidiate result from Branch to Dynamic Node %s for option %s stored as %s\n",
				fexec.id, currentNode.GetUniqueId(), option, key)
		}
		realIndegree, err := fexec.incrementCounter(branchkey, 1)
		if err != nil {
			return []byte(""), fmt.Errorf("failed to update inDegree counter for node %s", currentNode.GetUniqueId())
		}

		fexec.log("[Request `%s`] Executing end of dynamic node %s, completed indegree: %d/%d\n",
			fexec.id, currentNode.GetUniqueId(), realIndegree, len(options))

		//not last branch return
		if realIndegree < len(options) {
			return nil, nil
		}
	} else {
		fexec.log("[Request `%s`] Executing end of dynamic node %s, branch count 1\n",
			fexec.id, currentNode.GetUniqueId())
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
		fexec.log("[Request `%s`] Intermidiate result from Branch to Dynamic Node %s for option %s retrived from %s\n",
			fexec.id, currentNode.GetUniqueId(), option, key)
		// delete intermidiate data after retrival
		context.Del(key)

		subDataMap[option] = idata
	}

	// Get SubAggregator and call
	fexec.log("[Request `%s`] Executing aggregator of Dynamic Node %s\n",
		fexec.id, currentNode.GetUniqueId())
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
func (fexec *FlowExecutor) handleNextNodes(context *sdk.Context, result []byte) ([]byte, error) {
	// get pipeline
	pipeline := fexec.flow

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
			fexec.log("[Request `%s`] Intermidiate result from Node %s to %s stored as %s\n",
				fexec.id, currentNode.GetUniqueId(), node.GetUniqueId(), key)

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
			inDegreeUpdatedCount, err := fexec.incrementCounter(key, 1)
			if err != nil {
				return []byte(""), fmt.Errorf("failed to update inDegree counter for node %s", node.GetUniqueId())
			}

			// If all indegree has finished call that node
			if inDegree > inDegreeUpdatedCount {
				fexec.log("[Request `%s`] request for Node %s is delayed, completed indegree: %d/%d\n",
					fexec.id, node.GetUniqueId(), inDegreeUpdatedCount, inDegree)
				continue
			} else {
				fexec.log("[Request `%s`] performing request for Node %s, completed indegree: %d/%d\n",
					fexec.id, node.GetUniqueId(), inDegreeUpdatedCount, inDegree)
			}
		} else {
			fexec.log("[Request `%s`] performing request for Node %s, indegree count is 1\n",
				fexec.id, node.GetUniqueId())
		}
		// Set the DagExecutionPosition as of next Node
		pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_SAME, node.Id)

		// forward the flow request
		forwardErr := fexec.forwardState(currentNode.GetUniqueId(),
			intermediateData)
		if forwardErr != nil {
			// reset dag execution position
			pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_SAME, currentNode.Id)
			return nil, fmt.Errorf("Node(%s): error: %v",
				node.GetUniqueId(), forwardErr)
		}

		fexec.log("[Request `%s`] Async request submitted for Node %s\n",
			fexec.id, node.GetUniqueId())

	}

	// reset dag execution position
	pipeline.UpdatePipelineExecutionPosition(sdk.DEPTH_SAME, currentNode.Id)

	return []byte(""), nil
}

// handleFailure handles failure with failure handler and call finally
func (fexec *FlowExecutor) handleFailure(context *sdk.Context, err error) {
	var data []byte

	context.State = sdk.StateFailure
	// call failure handler if available
	if fexec.flow.FailureHandler != nil {
		fexec.log("[Request `%s`] Calling failure handler for error, %v\n",
			fexec.id, err)
		data, err = fexec.flow.FailureHandler(err)
	}

	fexec.finished = true

	// call finally handler if available
	if fexec.flow.Finally != nil {
		fexec.log("[Request `%s`] Calling Finally handler with state: %s\n",
			fexec.id, sdk.StateFailure)
		fexec.flow.Finally(sdk.StateFailure)
	}
	if data != nil {
		fexec.log("%s", string(data))
	}

	// Cleanup data and state for failure
	if fexec.stateStore != nil {
		fexec.stateStore.Cleanup()
	}
	fexec.dataStore.Cleanup()

	if fexec.executor.MonitoringEnabled() {
		fexec.eventHandler.ReportRequestFailure(fexec.id, err)
		fexec.eventHandler.Flush()
	}

	fmt.Sprintf("[Request `%s`] Failed, %v\n", fexec.id, err)
}

// getDagIntermediateData gets the intermediate data from earlier vertex
func (fexec *FlowExecutor) getDagIntermediateData(context *sdk.Context) ([]byte, error) {
	var data []byte

	pipeline := fexec.flow
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
			fexec.log("[Request `%s`] Intermidiate result from Node %s to Node %s for option %s retrived from %s\n",
				fexec.id, dagNode.GetUniqueId(), currentNode.GetUniqueId(),
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
			fexec.log("[Request `%s`] Intermidiate result from Node %s to Node %s retrived from %s\n",
				fexec.id, node.GetUniqueId(), currentNode.GetUniqueId(), key)
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
func (fexec *FlowExecutor) initializeStore() (stateSDefined bool, dataSOverride bool, err error) {

	stateSDefined = false
	dataSOverride = false

	// Initialize the stateS
	stateS, err := fexec.executor.GetStateStore()
	if err != nil {
		return
	}

	if stateS != nil {
		fexec.stateStore = stateS
		stateSDefined = true
		fexec.stateStore.Configure(fexec.flowName, fexec.id)
		// If request is not partial initialize the stateStore
		if !fexec.partial {
			err = fexec.stateStore.Init()
			if err != nil {
				return
			}
		}
	}

	// Initialize the dataS
	dataS, err := fexec.executor.GetDataStore()
	if err != nil {
		return
	}
	if dataS != nil {
		fexec.dataStore = dataS
		dataSOverride = true
	}
	fexec.dataStore.Configure(fexec.flowName, fexec.id)
	// If request is not partial initialize the dataStore
	if !fexec.partial {
		err = fexec.dataStore.Init()
	}

	return
}

// createContext create a context from request handler
func (fexec *FlowExecutor) createContext() *sdk.Context {
	context := sdk.CreateContext(fexec.id, "",
		fexec.flowName, fexec.dataStore)
	context.Query, _ = url.ParseQuery(fexec.query)

	return context
}

// init initialize the executor object and context
func (fexec *FlowExecutor) init() ([]byte, error) {
	requestId := ""
	var requestData []byte
	var err error

	switch fexec.partial {
	case false: // new request
		rawRequest := fexec.newRequest
		if fexec.executor.ReqAuthEnabled() {
			signature := rawRequest.AuthSignature
			key, err := fexec.executor.GetReqAuthKey()
			if err != nil {
				return nil, fmt.Errorf("failed to get auth key, error %v", err)
			}
			err = hmac.Validate(rawRequest.Data, signature, key)
			if err != nil {
				return nil, fmt.Errorf("failed to authenticate request, error %v", err)
			}
		}

		requestId = xid.New().String()
		fexec.executor.Configure(requestId)

		fexec.flowName = fexec.executor.GetFlowName()
		fexec.flow = sdk.CreatePipeline()
		fexec.id = requestId
		fexec.query = rawRequest.Query
		fexec.dataStore = createDataStore()

		requestData = rawRequest.Data

		if fexec.executor.MonitoringEnabled() {
			fexec.eventHandler, err = fexec.executor.GetEventHandler()
			if err != nil {
				return nil, fmt.Errorf("failed to initialize EventHandler, error %v", err)
			}
			fexec.eventHandler.Configure(fexec.flowName, fexec.id)
			err := fexec.eventHandler.Init()
			if err != nil {
				return nil, fmt.Errorf("failed to initialize EventHandler, error %v", err)
			}
			fexec.eventHandler.ReportRequestStart(fexec.id)
		}

		if fexec.executor.LoggingEnabled() {
			fexec.logger, err = fexec.executor.GetLogger()
			if err != nil {
				return nil, fmt.Errorf("failed to initiate logger, error %v", err)
			}
			fexec.logger.Configure(requestId, fexec.flowName)
			err = fexec.logger.Init()
			if err != nil {
				return nil, fmt.Errorf("failed to initiate logger, error %v", err)
			}
		}

		fexec.log("[Request `%s`] Created\n", requestId)

	case true: // partial request

		encodedState := fexec.partialState
		request, err := decodeRequest(encodedState)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse ExecutionState, error %v", err)
		}

		if fexec.executor.ReqValidationEnabled() {
			key, err := fexec.executor.GetValidationKey()
			if err != nil {
				return nil, fmt.Errorf("failed to get validation key, error %v", err)
			}
			err = hmac.Validate([]byte(request.ExecutionState), request.Sign, key)
			if err != nil {
				return nil, fmt.Errorf("failed to validate partial request, error %v", err)
			}
		}

		requestId = request.getID()
		fexec.executor.Configure(requestId)

		fexec.flowName = fexec.executor.GetFlowName()
		fexec.flow = sdk.CreatePipeline()
		fexec.flow.ApplyState(request.getExecutionState())
		fexec.id = requestId
		fexec.query = request.Query
		fexec.dataStore = retriveDataStore(request.getContextStore())

		requestData = request.getData()

		if fexec.executor.MonitoringEnabled() {
			fexec.eventHandler, err = fexec.executor.GetEventHandler()
			if err != nil {
				return nil, fmt.Errorf("failed to get EventHandler, error %v", err)
			}
			fexec.eventHandler.Configure(fexec.flowName, fexec.id)
			err := fexec.eventHandler.Init()
			if err != nil {
				return nil, fmt.Errorf("failed to initialize EventHandler, error %v", err)
			}
			fexec.eventHandler.ReportExecutionContinuation(fexec.id)
		}

		if fexec.executor.LoggingEnabled() {
			fexec.logger, err = fexec.executor.GetLogger()
			if err != nil {
				return nil, fmt.Errorf("failed to initiate logger, error %v", err)
			}
			fexec.logger.Configure(requestId, fexec.flowName)
			err = fexec.logger.Init()
			if err != nil {
				return nil, fmt.Errorf("failed to initiate logger, error %v", err)
			}
		}

		fexec.log("[Request `%s`] Received\n", requestId)

	}

	fexec.flowName = fexec.executor.GetFlowName()

	return requestData, nil
}

// applyExecutionState apply an execution state
func (fexec *FlowExecutor) applyExecutionState(state *ExecutionStateOptions) error {

	switch {
	case state.newRequest != nil:
		fexec.partial = false
		fexec.newRequest = state.newRequest

	case state.partialState != nil:
		fexec.partial = true
		fexec.partialState = state.partialState
	default:
		return fmt.Errorf("invalid execution state")
	}
	return nil
}

// GetReqId get request id
func (fexec *FlowExecutor) GetReqId() string {
	return fexec.id
}

// Execute start faasflow execution
func (fexec *FlowExecutor) Execute(state ExecutionStateOption) ([]byte, error) {
	var resp []byte
	var gerr error

	// Get State
	executionState := &ExecutionStateOptions{}
	state(executionState)
	err := fexec.applyExecutionState(executionState)
	if err != nil {
		return nil, err
	}

	// Init Flow: Create flow object
	data, err := fexec.init()
	if err != nil {
		return nil, err
	}

	// Init Stores: Get definition of StateStore and DataStore from user
	stateSDefined, dataSOverride, err := fexec.initializeStore()
	if err != nil {
		return nil, fmt.Errorf("[Request `%s`] Failed to init flow, %v", fexec.id, err)
	}

	// Make Context: make the request context from flow
	context := fexec.createContext()

	// Get Definition: Get Pipeline definition from user implemented Define()
	err = fexec.executor.GetFlowDefinition(fexec.flow, context)
	if err != nil {
		return nil, fmt.Errorf("[Request `%s`] Failed to define flow, %v", fexec.id, err)
	}

	// Validate Definition: Validate Pipeline Definition
	err = fexec.flow.Dag.Validate()
	if err != nil {
		return nil, fmt.Errorf("[Request `%s`] Invalid dag, %v", fexec.id, err)
	}

	// Check Configuration: Check if executor is properly configured to execute flow

	fexec.hasBranch = fexec.flow.Dag.HasBranch()
	fexec.hasEdge = fexec.flow.Dag.HasEdge()
	fexec.isExecutionFlow = fexec.flow.Dag.IsExecutionFlow()

	// For dag one of the branch can cause the pipeline to terminate
	// hence we Check if the pipeline is active
	if fexec.hasBranch && fexec.partial && !fexec.isActive() {
		return nil, fmt.Errorf("[Request `%s`] flow has been terminated", fexec.id)
	}

	// For dag which has branches
	// StateStore need to be external
	if fexec.hasBranch && !stateSDefined {
		return nil, fmt.Errorf("[Request `%s`] Failed, DAG flow need external StateStore", fexec.id)
	}

	// If dags has atleast one edge
	// and nodes forwards data, data store need to be external
	if fexec.hasEdge && !fexec.isExecutionFlow && !dataSOverride {
		return nil, fmt.Errorf("[Request `%s`] Failed not an execution flow, DAG data flow need external DataStore", fexec.id)
	}

	// Execute: Start execution

	// If not a partial request
	if !fexec.partial {

		// For a new dag pipeline that has branches Create the vertex in stateStore
		if fexec.hasBranch {
			serr := fexec.setRequestState(true)
			if serr != nil {
				return nil, fmt.Errorf("[Request `%s`] Failed to mark dag state, error %v", fexec.id, serr)
			}
			fexec.log("[Request `%s`] DAG state initiated at StateStore\n", fexec.id)
		}

		// set the execution position to initial node
		// On the 0th depth set the initial node as the current execution position
		fexec.flow.UpdatePipelineExecutionPosition(sdk.DEPTH_SAME,
			fexec.flow.GetInitialNodeId())
	}

	// if not an execution only dag, for partial request get intermediate data
	if fexec.partial && !fexec.flow.Dag.IsExecutionFlow() {

		// Get intermediate data from data store
		data, gerr = fexec.getDagIntermediateData(context)
		if gerr != nil {
			gerr := fmt.Errorf("failed to retrive intermediate result, error %v", gerr)
			fexec.log("[Request `%s`] Failed: %v\n", fexec.id, gerr)
			fexec.handleFailure(context, gerr)
			return nil, gerr
		}
	}

	// Find the right node to execute now
	fexec.findCurrentNodeToExecute()
	currentNode, _ := fexec.flow.GetCurrentNodeDag()
	result := []byte{}

	switch {
	// Execute the dynamic node
	case currentNode.Dynamic():
		result, err = fexec.executeDynamic(context, data)
		if err != nil {
			fexec.log("[Request `%s`] Failed: %v\n", fexec.id, err)
			fexec.handleFailure(context, err)
			return nil, err
		}
		// Execute the node
	default:
		result, err = fexec.executeNode(data)
		if err != nil {
			fexec.log("[Request `%s`] Failed: %v\n", fexec.id, err)
			fexec.handleFailure(context, err)
			return nil, err
		}

		// Find the right node to execute next
		fexec.findNextNodeToExecute()

	NodeCompletionLoop:
		for !fexec.finished {
			currentNode, _ := fexec.flow.GetCurrentNodeDag()
			switch {
			// Execute a end of dynamic node
			case currentNode.Dynamic():
				result, err = fexec.handleDynamicEnd(context, result)
				if err != nil {
					fexec.log("[Request `%s`] Failed: %v\n", fexec.id, err)
					fexec.handleFailure(context, err)
					return nil, err
				}
				// in case dynamic end can not be executed
				if result == nil {
					if fexec.executor.MonitoringEnabled() {
						fexec.eventHandler.ReportNodeEnd(currentNode.GetUniqueId(), fexec.id)
					}

					break NodeCompletionLoop
				}
				// in case dynamic nodes end has finished execution,
				// find next node and continue
				// although if next is a children handle it
				notLast := fexec.findNextNodeToExecute()
				if !notLast {
					continue
				}
				fallthrough
			default:
				// handle the execution iteration and update state
				result, err = fexec.handleNextNodes(context, result)
				if err != nil {
					fexec.log("[Request `%s`] Failed: %v\n", fexec.id, err)
					fexec.handleFailure(context, err)
					return nil, err
				}
				break NodeCompletionLoop
			}
		}
	}

	// Check if execution finished
	if fexec.finished {
		fexec.log("[Request `%s`] Completed successfully\n", fexec.id)
		context.State = sdk.StateSuccess
		if fexec.flow.Finally != nil {
			fexec.log("[Request `%s`] Calling Finally handler with state: %s\n",
				fexec.id, sdk.StateSuccess)
			fexec.flow.Finally(sdk.StateSuccess)
		}

		// Cleanup data and state for failure
		if fexec.stateStore != nil {
			fexec.stateStore.Cleanup()
		}
		fexec.dataStore.Cleanup()

		resp = result
	}

	if fexec.executor.MonitoringEnabled() {
		fexec.eventHandler.ReportRequestEnd(fexec.id)
		fexec.eventHandler.Flush()
	}

	return resp, nil
}

// CreateFlowExecutor initiate a FlowExecutor with a provided Executor
func CreateFlowExecutor(executor Executor) (fexec *FlowExecutor) {
	fexec = &FlowExecutor{}
	fexec.executor = executor

	return fexec
}
