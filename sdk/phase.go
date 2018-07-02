package sdk

type Phase struct {
	// The list of function in the Phase
	Functions []*Function `json:"functions"`
}

func CreateExecutionPhase() *Phase {
	phase := &Phase{}
	phase.Functions = make([]*Function, 0)
}

func (phase *Phase) AddFunction(function *Function) {
	append(phase.Functions, function)
}
