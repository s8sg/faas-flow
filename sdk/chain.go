package sdk

import (
	"encoding/json"
	"io"
)

type Chain struct {
	Phases            []*Phase `json:"phases"`   // Phases that will be executed in async
	ExecutionPosition int      `json:"position"` // Position of Executor
	Data              []byte   `json:"data"`     // The request data
}

func CreateChain() *Chain {
	chain = &Chain{}
	chain.Phases = make([]*Phase, 0)
	chain.ExecutionPosition = 0
	chain.Data = nil
	return chain
}

func (chain *Chain) AddPhase(phase *Phase) {
	append(chain.Phases, phase)
}

func (chain *Chain) GetLatestPhase(phase *Phase) Phase {
	phaseCount := len(chain.Phases)
	return fchain.chain.Phases[phaseCount-1]
}

func (chain *Chain) Encode() ([]byte, error) {
	return json.Marshal(chain)
}
