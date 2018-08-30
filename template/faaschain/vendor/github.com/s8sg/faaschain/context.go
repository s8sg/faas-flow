package faaschain

import (
	"net/url"
)

// Context execution context and execution state
type Context struct {
	requestId    string       // the request id
	phase        int          // the execution position
	stateManager StateManager // underline StateManager
	Query        url.Values   // provides request Query
	State        string       // state of the request
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
func CreateContext(id string, phase int) *Context {
	context := &Context{}
	context.requestId = id
	context.phase = phase
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

// Set put a value in the context with StateManager
func (context *Context) Set(key string, data interface{}) error {
	return context.stateManager.Set(key, data)
}

// Get retrive a value from the context with StateManager
func (context *Context) Get(key string) (interface{}, error) {
	data, err := context.stateManager.Get(key)
	return data, err
}

// Del deletes a value from the context with StateManager
func (context *Context) Del(key string) error {
	return context.stateManager.Del(key)
}
