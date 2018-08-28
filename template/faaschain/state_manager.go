package main

import (
	"fmt"
)

// json to encode
type requestEmbedStateManager struct {
	state map[string]interface{}
}

// CreateStateManager creates a new requestEmbedStateManager
func createStateManager() *requestEmbedStateManager {
	rstate := &requestEmbedStateManager{}
	rstate.state = make(map[string]interface{})
	return rstate
}

// RetriveStateManager creates a state manager from a map
func retriveStateManager(state map[string]interface{}) *requestEmbedStateManager {
	rstate := &requestEmbedStateManager{}
	rstate.state = state
	return rstate
}

// Set sets a value (implement StateManager)
func (rstate *requestEmbedStateManager) Set(key string, value interface{}) error {
	rstate.state[key] = value
	return nil
}

// Get gets a value (implement StateManager)
func (rstate *requestEmbedStateManager) Get(key string) (interface{}, error) {
	value, ok := rstate.state[key]
	if !ok {
		return nil, fmt.Errorf("No field name %s", key)
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
