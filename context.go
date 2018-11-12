package faasflow

import (
	"encoding/json"
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
	// Initialize the StateManager with flow name and request ID
	Init(flowName string, requestId string) error
	// Set store a value for key, in failure returns error
	Set(key string, value string) error
	// Get retrives a value by key, if failure returns error
	Get(key string) (string, error)
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

// SetStateManager sets and overwrite the state manager
func (context *Context) SetStateManager(state StateManager) error {
	context.stateManager = state
	err := context.stateManager.Init(context.Name, context.requestId)
	if err != nil {
		return fmt.Errorf("Failed to initialize StateManager, error %v", err)
	}
	return nil
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
	c := struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}{Key: key, Value: data}
	b, err := json.Marshal(&c)
	if err != nil {
		return fmt.Errorf("Failed to marshal data, error %v", err)
	}

	return context.stateManager.Set(key, string(b))
}

// Get retrive a value from the context using StateManager
func (context *Context) Get(key string) (interface{}, error) {
	data, err := context.stateManager.Get(key)
	if err != nil {
		return nil, err
	}
	c := struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal data, error %v", err)
	}
	return c.Value, err
}

// GetInt retrive a integer value from the context using StateManager
func (context *Context) GetInt(key string) (int, error) {
	data, err := context.stateManager.Get(key)
	if err != nil {
		return 0, err
	}

	c := struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		return 0, fmt.Errorf("Failed to unmarshal data, error %v", err)
	}

	intData, ok := c.Value.(int)
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

	c := struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal data, error %v", err)
	}

	stringData, ok := c.Value.(string)
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

	c := struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal data, error %v", err)
	}

	byteData, ok := c.Value.([]byte)
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

	c := struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		return false, fmt.Errorf("Failed to unmarshal data, error %v", err)
	}

	boolData, ok := c.Value.(bool)
	if !ok {
		return false, fmt.Errorf("failed to convert boolean for key %s", key)
	}
	return boolData, nil
}

// Del deletes a value from the context using StateManager
func (context *Context) Del(key string) error {
	return context.stateManager.Del(key)
}
