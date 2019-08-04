package faasflow

import (
	"fmt"
	sdk "github.com/s8sg/faas-flow/sdk"
)

type Context sdk.Context
type StateStore sdk.StateStore
type DataStore sdk.DataStore

// Options options for operation execution
type Options struct {
	// Operation options
	header          map[string]string
	query           map[string][]string
	failureHandler  FuncErrorHandler
	requestHandler  ReqHandler
	responseHandler RespHandler
}

// BranchOptions options for branching in DAG
type BranchOptions struct {
	aggregator  sdk.Aggregator
	forwarder   sdk.Forwarder
	noforwarder bool
}

type Workflow struct {
	pipeline *sdk.Pipeline // underline pipeline definition object
}

type Dag struct {
	udag *sdk.Dag
}

type Node struct {
	unode *sdk.Node
}

type Option func(*Options)
type BranchOption func(*BranchOptions)

var (
	// Execution specify a edge doesn't forwards a data
	// but rather mention a execution direction
	Execution = InvokeEdge()
)

// reset reset the Options
func (o *Options) reset() {
	o.header = map[string]string{}
	o.query = map[string][]string{}
	o.failureHandler = nil
	o.requestHandler = nil
	o.responseHandler = nil
}

// reset reset the BranchOptions
func (o *BranchOptions) reset() {
	o.aggregator = nil
	o.noforwarder = false
	o.forwarder = nil
}

// Aggregator aggregates all outputs into one
func Aggregator(aggregator sdk.Aggregator) BranchOption {
	return func(o *BranchOptions) {
		o.aggregator = aggregator
	}
}

// InvokeEdge denotes a edge doesn't forwards a data,
// but rather provides only an execution flow
func InvokeEdge() BranchOption {
	return func(o *BranchOptions) {
		o.noforwarder = true
	}
}

// Forwarder encodes request based on need for children vertex
// by default the data gets forwarded as it is
func Forwarder(forwarder sdk.Forwarder) BranchOption {
	return func(o *BranchOptions) {
		o.forwarder = forwarder
	}
}

// Header Specify a header in a http call
func Header(key, value string) Option {
	return func(o *Options) {
		o.header[key] = value
	}
}

// Query Specify a query parameter in a http call
func Query(key string, value ...string) Option {
	return func(o *Options) {
		array := []string{}
		for _, val := range value {
			array = append(array, val)
		}
		o.query[key] = array
	}
}

// OnFailure Specify a function failure handler
func OnFailure(handler FuncErrorHandler) Option {
	return func(o *Options) {
		o.failureHandler = handler
	}
}

// RequestHandler Specify a request handler for function and callback request
func RequestHandler(handler ReqHandler) Option {
	return func(o *Options) {
		o.requestHandler = handler
	}
}

// OnResponse Specify a response handler for function and callback
func OnReponse(handler RespHandler) Option {
	return func(o *Options) {
		o.responseHandler = handler
	}
}

// GetWorkflow initiates a flow with a pipeline
func GetWorkflow(pipeline *sdk.Pipeline) *Workflow {
	workflow := &Workflow{}
	workflow.pipeline = pipeline
	return workflow
}

// OnFailure set a failure handler routine for the pipeline
func (flow *Workflow) OnFailure(handler sdk.PipelineErrorHandler) {
	flow.pipeline.FailureHandler = handler
}

// Finally sets an execution finish handler routine
// it will be called once the execution has finished with state either Success/Failure
func (flow *Workflow) Finally(handler sdk.PipelineHandler) {
	flow.pipeline.Finally = handler
}

// GetPipeline expose the underlying pipeline object
func (flow *Workflow) GetPipeline() *sdk.Pipeline {
	return flow.pipeline
}

// Dag provides the workflow dag object
func (flow *Workflow) Dag() *Dag {
	dag := &Dag{}
	dag.udag = flow.pipeline.Dag
	return dag
}

// SetDag apply a predefined dag, and override the default dag
func (flow *Workflow) SetDag(dag *Dag) {
	pipeline := flow.pipeline
	pipeline.SetDag(dag.udag)
}

// NewDag creates a new dag seperately from pipeline
func NewDag() *Dag {
	dag := &Dag{}
	dag.udag = sdk.NewDag()
	return dag
}

// Append generalizes a seperate dag by appending its properties into current dag.
// Provided dag should be mutually exclusive
func (this *Dag) Append(dag *Dag) {
	err := this.udag.Append(dag.udag)
	if err != nil {
		panic(fmt.Sprintf("Error at AppendDag, %v", err))
	}
}

// Node adds a new vertex by id
func (this *Dag) Node(vertex string, options ...BranchOption) *Node {
	node := this.udag.GetNode(vertex)
	if node == nil {
		node = this.udag.AddVertex(vertex, []sdk.Operation{})
	}
	o := &BranchOptions{}
	for _, opt := range options {
		o.reset()
		opt(o)
		if o.aggregator != nil {
			node.AddAggregator(o.aggregator)
		}
	}
	return &Node{unode: node}
}

// Edge adds a directed edge between two vertex as <from>-><to>
func (this *Dag) Edge(from, to string, opts ...BranchOption) {
	err := this.udag.AddEdge(from, to)
	if err != nil {
		panic(fmt.Sprintf("Error at AddEdge for %s-%s, %v", from, to, err))
	}
	o := &BranchOptions{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if o.noforwarder == true {
			fromNode := this.udag.GetNode(from)
			// Add a nil forwarder overriding the default forwarder
			fromNode.AddForwarder(to, nil)
		}

		// in case there is a override
		if o.forwarder != nil {
			fromNode := this.udag.GetNode(from)
			fromNode.AddForwarder(to, o.forwarder)
		}
	}
}

// SubDag composites a seperate dag as a node.
func (this *Dag) SubDag(vertex string, dag *Dag) {
	node := this.udag.AddVertex(vertex, []sdk.Operation{})
	err := node.AddSubDag(dag.udag)
	if err != nil {
		panic(fmt.Sprintf("Error at AddSubDag for %s, %v", vertex, err))
	}
	return
}

// ForEachBranch composites a subdag which executes for each value
// It returns the subdag that will be executed for each value
func (this *Dag) ForEachBranch(vertex string, foreach sdk.ForEach, options ...BranchOption) (dag *Dag) {
	node := this.udag.AddVertex(vertex, []sdk.Operation{})
	if foreach == nil {
		panic(fmt.Sprintf("Error at AddForEachBranch for %s, foreach function not specified", vertex))
	}
	node.AddForEach(foreach)

	for _, option := range options {
		o := &BranchOptions{}
		o.reset()
		option(o)
		if o.aggregator != nil {
			node.AddSubAggregator(o.aggregator)
		}
		if o.noforwarder == true {
			node.AddForwarder("dynamic", nil)
		}
	}

	dag = NewDag()
	err := node.AddForEachDag(dag.udag)
	if err != nil {
		panic(fmt.Sprintf("Error at AddForEachBranch for %s, %v", vertex, err))
	}
	return
}

// ConditionalBranch composites multiple dags as a subdag which executes for a conditions matched
// and returns the set of dags based on the condition passed
func (this *Dag) ConditionalBranch(vertex string, conditions []string, condition sdk.Condition,
	options ...BranchOption) (conditiondags map[string]*Dag) {

	node := this.udag.AddVertex(vertex, []sdk.Operation{})
	if condition == nil {
		panic(fmt.Sprintf("Error at AddConditionalBranch for %s, condition function not specified", vertex))
	}
	node.AddCondition(condition)

	for _, option := range options {
		o := &BranchOptions{}
		o.reset()
		option(o)
		if o.aggregator != nil {
			node.AddSubAggregator(o.aggregator)
		}
		if o.noforwarder == true {
			node.AddForwarder("dynamic", nil)
		}
	}
	conditiondags = make(map[string]*Dag)
	for _, conditionKey := range conditions {
		dag := NewDag()
		node.AddConditionalDag(conditionKey, dag.udag)
		conditiondags[conditionKey] = dag
	}
	return
}

// Modify adds a new modifier to the given vertex
func (node *Node) Modify(mod Modifier) *Node {
	newMod := createModifier(mod)
	node.unode.AddOperation(newMod)
	return node
}

// Apply adds a new function to the given vertex
func (node *Node) Apply(function string, opts ...Option) *Node {

	newfunc := createFunction(function)

	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if len(o.header) != 0 {
			for key, value := range o.header {
				newfunc.addheader(key, value)
			}
		}
		if len(o.query) != 0 {
			for key, array := range o.query {
				for _, value := range array {
					newfunc.addparam(key, value)
				}
			}
		}
		if o.failureHandler != nil {
			newfunc.addFailureHandler(o.failureHandler)
		}
		if o.responseHandler != nil {
			newfunc.addResponseHandler(o.responseHandler)
		}
		if o.requestHandler != nil {
			newfunc.addRequestHandler(o.requestHandler)
		}
	}

	node.unode.AddOperation(newfunc)
	return node
}

// Callback adds a new callback to the given vertex
func (node *Node) Callback(url string, opts ...Option) *Node {
	newCallback := createCallback(url)

	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if len(o.header) != 0 {
			for key, value := range o.header {
				newCallback.addheader(key, value)
			}
		}
		if len(o.query) != 0 {
			for key, array := range o.query {
				for _, value := range array {
					newCallback.addparam(key, value)
				}
			}
		}
		if o.failureHandler != nil {
			newCallback.addFailureHandler(o.failureHandler)
		}
		if o.responseHandler != nil {
			newCallback.addResponseHandler(o.responseHandler)
		}
		if o.requestHandler != nil {
			newCallback.addRequestHandler(o.requestHandler)
		}
	}

	node.unode.AddOperation(newCallback)
	return node
}

// SyncNode adds a new vertex named Sync
func (flow *Workflow) SyncNode(options ...BranchOption) *Node {

	dag := flow.pipeline.Dag

	node := dag.GetNode("sync")
	if node == nil {
		node = dag.AddVertex("sync", []sdk.Operation{})
	}
	o := &BranchOptions{}
	for _, opt := range options {
		o.reset()
		opt(o)
		if o.aggregator != nil {
			node.AddAggregator(o.aggregator)
		}
	}
	return &Node{unode: node}
}
