package faasflow

import (
	"fmt"
	"net/url"
)

// Context execution context and execution state
type Context struct {
	requestId    string       // the request id
	phase        int          // the execution position
	stateManager StateManager // underline StateManager
	Query        url.Values   // provides request Query
	State        string       // state of the request
	Name         string
}

// StateManager for State Manager
type StateManager interface {
	// Set store a value for key, in failure returns error
	Set(key string, value interface{}) error
	// Get retrives a value by key, if failure returns error
	Get(key string) (interface{}, error)
	// Del delets a value by a key
	Del(key string) error
}

const (
	// StateSuccess denotes success state
	StateSuccess = "success"
	// StateFailure denotes failure state
	StateFailure = "failure"
	// StateOngoing denotes onging satte
	StateOngoing = "ongoing"
)

// CreateContext create request context (used by template)
func CreateContext(id string, phase int, name string) *Context {
	context := &Context{}
	context.requestId = id
	context.phase = phase
	context.Name = name
	context.State = StateOngoing

	return context
}

// SetStateManager sets the state manager
func (context *Context) SetStateManager(state StateManager) {
	context.stateManager = state
}

// GetRequestId returns the request id
func (context *Context) GetRequestId() string {
	return context.requestId
}

// GetPhase return the phase no
func (context *Context) GetPhase() int {
	return context.phase
}

// Set put a value in the context using StateManager
func (context *Context) Set(key string, data interface{}) error {
	return context.stateManager.Set(key, data)
}

// Get retrive a value from the context using StateManager
func (context *Context) Get(key string) (interface{}, error) {
	data, err := context.stateManager.Get(key)
	return data, err
}

// GetInt retrive a integer value from the context using StateManager
func (context *Context) GetInt(key string) (int, error) {
	data, err := context.stateManager.Get(key)
	if err != nil {
		return 0, err
	}
	intData, ok := data.(int)
	if !ok {
		return 0, fmt.Errorf("failed to convert int for key %s", key)
	}
	return intData, nil
}

// GetString retrive a string value from the context using StateManager
func (context *Context) GetString(key string) (string, error) {
	data, err := context.stateManager.Get(key)
	if err != nil {
		return "", err
	}
	stringData, ok := data.(string)
	if !ok {
		return "", fmt.Errorf("failed to convert string for key %s", key)
	}
	return stringData, nil
}

// GetBytes retrive a byte array from the context using StateManager
func (context *Context) GetBytes(key string) ([]byte, error) {
	data, err := context.stateManager.Get(key)
	if err != nil {
		return nil, err
	}
	byteData, ok := data.([]byte)
	if !ok {
		return nil, fmt.Errorf("failed to convert byte array for key %s", key)
	}
	return byteData, nil
}

// GetBool retrive a boolean value from the context using StateManager
func (context *Context) GetBool(key string) (bool, error) {
	data, err := context.stateManager.Get(key)
	if err != nil {
		return false, err
	}
	boolData, ok := data.(bool)
	if !ok {
		return false, fmt.Errorf("failed to convert boolean for key %s", key)
	}
	return boolData, nil
}

// Del deletes a value from the context using StateManager
func (context *Context) Del(key string) error {
	return context.stateManager.Del(key)
}
