package sdk

type Phase struct {
	// The list of function in the Phase
	Operations []*Operation
}

func CreateExecutionPhase() *Phase {
	phase := &Phase{}
	phase.Operations = make([]*Operation, 0)
	return phase
}

func (phase *Phase) AddOperation(function *Operation) {
	phase.Operations = append(phase.Operations, function)
}

func (phase *Phase) GetOperations() []*Operation {
	return phase.Operations
}
