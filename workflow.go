package faasflow

import (
	"fmt"
	"github.com/s8sg/faas-flow/sdk"
)

// Options options for operation execution
type Options struct {
	// Operation options
	header          map[string]string
	query           map[string][]string
	failureHandler  sdk.FuncErrorHandler
	requestHandler  sdk.ReqHandler
	responseHandler sdk.RespHandler

	// Operation Chaining options
	sync bool
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

type DagFlow struct {
	udag *sdk.Dag
}

type Option func(*Options)
type BranchOption func(*BranchOptions)

var (
	// Sync can be used instead of SyncCall
	Sync = SyncCall()
	// Execution specify a edge doesn't forwards a data
	// but rather mention a execution direction
	Execution = InvokeEdge()
	// Denote if last node doesn't contain any function call
	emptyNode = false
	// the reference of lastnode when applied as chain
	lastnode *sdk.Node = nil
)

// reset reset the Options
func (o *Options) reset() {
	o.header = map[string]string{}
	o.query = map[string][]string{}
	o.sync = false
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

// SyncCall Set sync flag, denotes a call to be in sync
func SyncCall() Option {
	return func(o *Options) {
		o.sync = true
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
func OnFailure(handler sdk.FuncErrorHandler) Option {
	return func(o *Options) {
		o.failureHandler = handler
	}
}

// RequestHdlr Specify a request handler for function and callback
func RequestHdlr(handler sdk.ReqHandler) Option {
	return func(o *Options) {
		o.requestHandler = handler
	}
}

// OnResponse Specify a response handler for function and callback
func OnReponse(handler sdk.RespHandler) Option {
	return func(o *Options) {
		o.responseHandler = handler
	}
}

// Modify allows to apply inline callback function
// the callback fucntion prototype is
// func([]byte) ([]byte, error)
func (flow *Workflow) Modify(mod sdk.Modifier) *Workflow {
	newMod := sdk.CreateModifier(mod)

	if flow.pipeline.CountNodes() == 0 {
		id := fmt.Sprintf("%d", flow.pipeline.CountNodes())
		// create a new vertex and add modifier
		flow.pipeline.Dag.AddVertex(id, []*sdk.Operation{newMod})
		lastnode = flow.pipeline.Dag.GetNode(id)
		emptyNode = true
	} else {
		node := lastnode
		node.AddOperation(newMod)
	}
	return flow
}

// Callback register a callback url as a part of pipeline definition
// One or more callback function can be placed for sending
// either partial pipeline data or after the pipeline completion
func (flow *Workflow) Callback(url string, opts ...Option) *Workflow {
	newCallback := sdk.CreateCallback(url)

	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if len(o.header) != 0 {
			for key, value := range o.header {
				newCallback.Addheader(key, value)
			}
		}
		if len(o.query) != 0 {
			for key, array := range o.query {
				for _, value := range array {
					newCallback.Addparam(key, value)
				}
			}
		}
		if o.failureHandler != nil {
			newCallback.AddFailureHandler(o.failureHandler)
		}
	}

	if flow.pipeline.CountNodes() == 0 {
		id := fmt.Sprintf("%d", flow.pipeline.CountNodes())
		// create a new vertex and add callback
		flow.pipeline.Dag.AddVertex(id, []*sdk.Operation{newCallback})
		lastnode = flow.pipeline.Dag.GetNode(id)
		emptyNode = true
	} else {
		node := lastnode
		node.AddOperation(newCallback)
	}
	return flow
}

// Apply apply a function with given name and options
// default call is async, provide Sync option to call synchronously
func (flow *Workflow) Apply(function string, opts ...Option) *Workflow {
	newfunc := sdk.CreateFunction(function)
	sync := false

	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if len(o.header) != 0 {
			for key, value := range o.header {
				newfunc.Addheader(key, value)
			}
		}
		if len(o.query) != 0 {
			for key, array := range o.query {
				for _, value := range array {
					newfunc.Addparam(key, value)
				}
			}
		}
		if o.sync {
			sync = true
		}
		if o.failureHandler != nil {
			newfunc.AddFailureHandler(o.failureHandler)
		}
		if o.responseHandler != nil {
			newfunc.AddResponseHandler(o.responseHandler)
		}
		if o.requestHandler != nil {
			newfunc.AddRequestHandler(o.requestHandler)
		}
	}

	if sync {
		if flow.pipeline.CountNodes() == 0 {
			id := fmt.Sprintf("%d", flow.pipeline.CountNodes())
			// create a new vertex and add function
			flow.pipeline.Dag.AddVertex(id, []*sdk.Operation{newfunc})
			lastnode = flow.pipeline.Dag.GetNode(id)
		} else {
			node := lastnode
			node.AddOperation(newfunc)
		}
	} else {
		// If last node doesn't have any function its emptyNode
		if emptyNode {
			node := lastnode
			node.AddOperation(newfunc)
		} else {
			from := lastnode.Id
			id := fmt.Sprintf("%d", flow.pipeline.CountNodes())
			// create a new vertex and add function
			flow.pipeline.Dag.AddVertex(id, []*sdk.Operation{newfunc})
			lastnode = flow.pipeline.Dag.GetNode(id)
			flow.pipeline.Dag.AddEdge(from, id)
		}
	}
	emptyNode = false

	return flow
}

// ExecuteDag apply a predefined dag
// All operation inside dag are async
// returns error is dag is not valid
// Note: If executing dag chain gets overridden
func (flow *Workflow) ExecuteDag(dag *DagFlow) {
	pipeline := flow.pipeline
	pipeline.SetDag(dag.udag)
}

// OnFailure set a failure handler routine for the pipeline
func (flow *Workflow) OnFailure(handler sdk.PipelineErrorHandler) *Workflow {
	flow.pipeline.FailureHandler = handler
	return flow
}

// Finally sets an execution finish handler routine
// it will be called once the execution has finished with state either Success/Failure
func (flow *Workflow) Finally(handler sdk.PipelineHandler) *Workflow {
	flow.pipeline.Finally = handler
	return flow
}

// NewFaasflow initiates a flow with a pipeline
func NewFaasflow(name string) *Workflow {
	flow := &Workflow{}
	flow.pipeline = sdk.CreatePipeline(name)
	return flow
}

// GetPipeline expose the underlying pipeline object
func (flow *Workflow) GetPipeline() *sdk.Pipeline {
	return flow.pipeline
}
