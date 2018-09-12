package faasflow

import (
	"github.com/s8sg/faasflow/sdk"
)

type Options struct {
	header          map[string]string
	query           map[string][]string
	sync            bool
	failureHandler  sdk.FuncErrorHandler
	responseHandler sdk.RespHandler
}

type Workflow struct {
	pipeline *sdk.Pipeline // underline pipeline definition object
}

type Option func(*Options)

var (
	// Sync can be used instead of SyncCall
	Sync = SyncCall()
	// Denote if last phase doesn't contain any function
	emptyPhase = false
)

// reset reset the options
func (o *Options) reset() {
	o.header = map[string]string{}
	o.query = map[string][]string{}
	o.sync = false
	o.failureHandler = nil
	o.responseHandler = nil
}

// SyncCall Set sync flag
func SyncCall() Option {
	return func(o *Options) {
		o.sync = true
	}
}

// Header Specify a header
func Header(key, value string) Option {
	return func(o *Options) {
		o.header[key] = value
	}
}

// Query Specify a query parameter
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

// OnResponse Specify a resp handler callback for function
func OnReponse(handler sdk.RespHandler) Option {
	return func(o *Options) {
		o.responseHandler = handler
	}
}

// Modify allows to apply inline callback functionl
// the callback fucntion prototype is
// func([]byte) ([]byte, error)
func (flow *Workflow) Modify(mod sdk.Modifier) *Workflow {
	var phase *sdk.Phase
	newMod := sdk.CreateModifier(mod)
	if len(flow.pipeline.Phases) == 0 {
		phase = sdk.CreateExecutionPhase()
		flow.pipeline.AddPhase(phase)
		emptyPhase = true
	} else {
		phase = flow.pipeline.GetLastPhase()
	}
	phase.AddFunction(newMod)
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

	var phase *sdk.Phase
	if len(flow.pipeline.Phases) == 0 {
		phase = sdk.CreateExecutionPhase()
		flow.pipeline.AddPhase(phase)
		emptyPhase = true
	} else {
		phase = flow.pipeline.GetLastPhase()
	}
	phase.AddFunction(newCallback)
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
	}

	var phase *sdk.Phase
	if sync {
		if len(flow.pipeline.Phases) == 0 {
			phase = sdk.CreateExecutionPhase()
			flow.pipeline.AddPhase(phase)
		} else {
			phase = flow.pipeline.GetLastPhase()
		}
	} else {
		if emptyPhase {
			phase = flow.pipeline.GetLastPhase()
		} else {
			phase = sdk.CreateExecutionPhase()
			flow.pipeline.AddPhase(phase)
		}
	}
	emptyPhase = false
	phase.AddFunction(newfunc)

	return flow
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
func NewFaasflow() *Workflow {
	flow := &Workflow{}
	flow.pipeline = sdk.CreatePipeline()
	return flow
}

// GetPipeline expose the underlying pipeline object
func (flow *Workflow) GetPipeline() *sdk.Pipeline {
	return flow.pipeline
}
