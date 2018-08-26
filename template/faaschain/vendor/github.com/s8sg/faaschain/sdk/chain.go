package sdk

import (
	"encoding/json"
)

type Handler func(error)

type Chain struct {
	Phases            []*Phase `json:"-"`        // Phases that will be executed in async
	ExecutionPosition int      `json:"position"` // Position of Executor
	FailureHandler    Handler  `json:"-"`
}

func CreateChain() *Chain {
	chain := &Chain{}
	chain.Phases = make([]*Phase, 0)
	chain.ExecutionPosition = 0
	return chain
}

func DecodeChain(data []byte) (*Chain, error) {
	chain := &Chain{}
	err := json.Unmarshal(data, chain)
	if err != nil {
		return nil, err
	}
	return chain, nil
}

func (chain *Chain) CountPhases() int {
	return len(chain.Phases)
}

func (chain *Chain) AddPhase(phase *Phase) {
	chain.Phases = append(chain.Phases, phase)
}

func (chain *Chain) GetCurrentPhase() *Phase {
	if chain.ExecutionPosition < len(chain.Phases) {
		return chain.Phases[chain.ExecutionPosition]
	} else {
		return nil
	}
}

func (chain *Chain) IsLastPhase() bool {
	return ((len(chain.Phases) - 1) == chain.ExecutionPosition)
}

func (chain *Chain) GetLastPhase() *Phase {
	phaseCount := len(chain.Phases)
	return chain.Phases[phaseCount-1]
}

func (chain *Chain) UpdateExecutionPosition() {
	chain.ExecutionPosition = chain.ExecutionPosition + 1
}

func (chain *Chain) Encode() ([]byte, error) {
	return json.Marshal(chain)
}
