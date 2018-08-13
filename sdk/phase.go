package sdk

type Phase struct {
	// The list of function in the Phase
	Functions []*Function `json:"-"`
}

func CreateExecutionPhase() *Phase {
	phase := &Phase{}
	phase.Functions = make([]*Function, 0)
	return phase
}

func (phase *Phase) AddFunction(function *Function) {
	phase.Functions = append(phase.Functions, function)
}

func (phase *Phase) GetFunctions() []*Function {
	return phase.Functions
}
