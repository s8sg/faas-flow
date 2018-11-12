package main

import (
	"fmt"
)

// json to encode
type requestEmbedStateManager struct {
	state map[string]string
}

// CreateStateManager creates a new requestEmbedStateManager
func createStateManager() *requestEmbedStateManager {
	rstate := &requestEmbedStateManager{}
	rstate.state = make(map[string]string)
	return rstate
}

// RetriveStateManager creates a state manager from a map
func retriveStateManager(state map[string]string) *requestEmbedStateManager {
	rstate := &requestEmbedStateManager{}
	rstate.state = state
	return rstate
}

// Init initialize the statemanager with requestId and flowname
func (rstate *requestEmbedStateManager) Init(flowName string, requestId string) error {
	return nil
}

// Set sets a value (implement StateManager)
func (rstate *requestEmbedStateManager) Set(key string, value string) error {
	rstate.state[key] = value
	return nil
}

// Get gets a value (implement StateManager)
func (rstate *requestEmbedStateManager) Get(key string) (string, error) {
	value, ok := rstate.state[key]
	if !ok {
		return "", fmt.Errorf("No field name %s", key)
	}
	return value, nil
}

// Del delets a value (implement StateManager)
func (rstate *requestEmbedStateManager) Del(key string) error {
	if _, ok := rstate.state[key]; ok {
		delete(rstate.state, key)
	}
	return nil
}
