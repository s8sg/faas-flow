package faaschain

import (
	"github.com/s8sg/faaschain/sdk"
	"net/url"
	"path"
)

type Fchain struct {
	chain    *sdk.Chain
	id       string
	url      string
	asyncUrl string
	chainDef []byte
}

type Context struct {
	phaseInput []byte
	requestId  string
	phase      int
}

var (
	gContext *Context
)

// NewFaaschain creates a new faaschain object
func NewFaaschain(gateway string, chain string) *Fchain {
	fchain := &Fchain{}
	fchain.chain = sdk.CreateChain()
	u, _ := url.Parse(gateway)
	u.Path = path.Join(u.Path, "function/"+chain)
	fchain.url = u.String()
	u, _ = url.Parse(gateway)
	u.Path = path.Join(u.Path, "async-function/"+chain)
	fchain.asyncUrl = u.String()
	return fchain
}

// SetId set request id to a chain
func (fchain *Fchain) SetId(id string) {
	fchain.id = id
}

// ApplyModifier allows to apply inline callback functionl
// the callback fucntion prototype is
// func([]byte) ([]byte, error)
func (fchain *Fchain) ApplyModifier(mod sdk.Modifier) *Fchain {
	var phase *sdk.Phase
	newMod := sdk.CreateModifier(mod)
	if len(fchain.chain.Phases) == 0 {
		phase = sdk.CreateExecutionPhase()
		fchain.chain.AddPhase(phase)
	} else {
		phase = fchain.chain.GetLastPhase()
	}
	phase.AddFunction(newMod)
	return fchain
}

// Callback register a callback url as a part of chain definition
// One or more callback function can be placed for sending
// either partial chain data or after the chain completion
func (fchain *Fchain) Callback(url string, header map[string]string, param map[string][]string) *Fchain {
	newCallback := sdk.CreateCallback(url)
	if header != nil {
		for key, value := range header {
			newCallback.Addheader(key, value)
		}
	}
	if param != nil {
		for key, array := range param {
			for _, value := range array {
				newCallback.Addparam(key, value)
			}
		}
	}
	var phase *sdk.Phase
	if len(fchain.chain.Phases) == 0 {
		phase = sdk.CreateExecutionPhase()
		fchain.chain.AddPhase(phase)
	} else {
		phase = fchain.chain.GetLastPhase()
	}
	phase.AddFunction(newCallback)
	return fchain
}

// ApplyFunction apply a function of type sdk.Function
func (fchain *Fchain) ApplyFunction(function *sdk.Function) *Fchain {
	var phase *sdk.Phase
	if len(fchain.chain.Phases) == 0 {
		phase = sdk.CreateExecutionPhase()
		fchain.chain.AddPhase(phase)
	} else {
		phase = fchain.chain.GetLastPhase()
	}
	phase.AddFunction(function)
	return fchain
}

// Apply apply a function with given name and options
func (fchain *Fchain) Apply(function string, header map[string]string, param map[string][]string) *Fchain {
	newfunc := sdk.CreateFunction(function)
	if header != nil {
		for key, value := range header {
			newfunc.Addheader(key, value)
		}
	}
	if param != nil {
		for key, array := range param {
			for _, value := range array {
				newfunc.Addparam(key, value)
			}
		}
	}
	var phase *sdk.Phase
	if len(fchain.chain.Phases) == 0 {
		phase = sdk.CreateExecutionPhase()
		fchain.chain.AddPhase(phase)
	} else {
		phase = fchain.chain.GetLastPhase()
	}
	phase.AddFunction(newfunc)
	return fchain
}

// ApplyFunctionAsync apply a function of type sdk.Function which will be called
// in Async
// async apply creates a new phase
func (fchain *Fchain) ApplyFunctionAsync(function *sdk.Function) *Fchain {
	phase := sdk.CreateExecutionPhase()
	fchain.chain.AddPhase(phase)
	phase.AddFunction(function)
	return fchain
}

// ApplyAsync apply a function with given name and options which will be called
// in Async
// async apply creates a new phase
func (fchain *Fchain) ApplyAsync(function string, header map[string]string, param map[string][]string) *Fchain {
	newfunc := sdk.CreateFunction(function)
	if header != nil {
		for key, value := range header {
			newfunc.Addheader(key, value)
		}
	}
	if param != nil {
		for key, array := range param {
			for _, value := range array {
				newfunc.Addparam(key, value)
			}
		}
	}
	phase := sdk.CreateExecutionPhase()
	fchain.chain.AddPhase(phase)
	phase.AddFunction(newfunc)
	return fchain
}

// Build encode a underline faaschain
func (fchain *Fchain) Build() (err error) {
	fchain.chainDef, err = fchain.chain.Encode()
	return err
}

// GetDefinition provide definition of chain
func (fchain *Fchain) GetDefinition() string {
	return string(fchain.chainDef)
}

// GetChain provides the underlying chain object
func (fchain *Fchain) GetChain() *sdk.Chain {
	return fchain.chain
}

// GetId returns the current chain id
func (fchain *Fchain) GetId() string {
	return fchain.id
}

// GetUrl returns the URL for the faaschain function
func (fchain *Fchain) GetUrl() string {
	return fchain.url
}

// GetAsyncUrl returns the URL for the faaschain async function
func (fchain *Fchain) GetAsyncUrl() string {
	return fchain.asyncUrl
}

// CreateGlobalContext create a context for the chain
func (fchain *Fchain) CreateGlobalContext(request []byte) {
	context := &Context{}
	context.phaseInput = request
	context.requestId = fchain.id
	context.phase = fchain.chain.ExecutionPosition + 1
	gContext = context
}

// GetContext returns the global context that was created
func GetContext() *Context {
	return gContext
}

// GetPhaseInput returns the phase input (it allows to user replay a data )
func (context *Context) GetPhaseInput() []byte {
	return context.phaseInput
}

// GetRequestId returns the request id
func (context *Context) GetRequestId() string {
	return context.requestId
}

// GetPhase return the phase no
func (context *Context) GetPhase() int {
	return context.phase
}
