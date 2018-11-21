package faasflow

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// Context execution context and execution state
type Context struct {
	requestId string     // the request id
	phase     int        // the execution position
	dataStore DataStore  // underline DataStore
	Query     url.Values // provides request Query
	State     string     // state of the request
	Name      string
}

// DataStore for Storing Data
type DataStore interface {
	// Configure the DaraStore with flow name and request ID
	Configure(flowName string, requestId string)
	// Initialize the DataStore (called only once in a request span)
	Init() error
	// Set store a value for key, in failure returns error
	Set(key string, value string) error
	// Get retrives a value by key, if failure returns error
	Get(key string) (string, error)
	// Del delets a value by a key
	Del(key string) error
	// Cleanup all the resorces in DataStore
	Cleanup() error
}

// StateStore for saving execution state
type StateStore interface {
	// Configure the StateStore with flow name and request ID
	Configure(flowName string, requestId string)
	// Initialize the StateStore (called only once in a request span)
	Init() error
	// create Vertexes for request
	// creates a map[<vertexId>]<Indegree Completion Count>
	Create(vertexs []string) error
	// Increment Vertex Indegree Completion
	// synchronously increment map[<vertexId>] Indegree Completion Count by 1 and return updated count
	IncrementCounter(vertex string) (int, error)
	// Set state of pipeline
	SetState(state bool) error
	// Get State of pipeline
	GetState() (bool, error)
	// Cleanup all the resorces in StateStore (called only once in a request span)
	Cleanup() error
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
func CreateContext(id string, phase int, name string, dstore DataStore) *Context {
	context := &Context{}
	context.requestId = id
	context.phase = phase
	context.Name = name
	context.State = StateOngoing
	context.dataStore = dstore

	return context
}

// GetRequestId returns the request id
func (context *Context) GetRequestId() string {
	return context.requestId
}

// GetPhase return the phase no
func (context *Context) GetPhase() int {
	return context.phase
}

// Set put a value in the context using DataStore
func (context *Context) Set(key string, data interface{}) error {
	c := struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}{Key: key, Value: data}
	b, err := json.Marshal(&c)
	if err != nil {
		return fmt.Errorf("Failed to marshal data, error %v", err)
	}

	return context.dataStore.Set(key, string(b))
}

// Get retrive a value from the context using DataStore
func (context *Context) Get(key string) (interface{}, error) {
	data, err := context.dataStore.Get(key)
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

// GetInt retrive a integer value from the context using DataStore
func (context *Context) GetInt(key string) (int, error) {
	data, err := context.dataStore.Get(key)
	if err != nil {
		return 0, err
	}

	c := struct {
		Key   string `json:"key"`
		Value int    `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		return 0, fmt.Errorf("Failed to unmarshal data, error %v", err)
	}

	return c.Value, nil
}

// GetString retrive a string value from the context using DataStore
func (context *Context) GetString(key string) (string, error) {
	data, err := context.dataStore.Get(key)
	if err != nil {
		return "", err
	}

	c := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal data, error %v", err)
	}

	return c.Value, nil
}

// GetBytes retrive a byte array from the context using DataStore
func (context *Context) GetBytes(key string) ([]byte, error) {
	data, err := context.dataStore.Get(key)
	if err != nil {
		return nil, err
	}

	c := struct {
		Key   string `json:"key"`
		Value []byte `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal data, error %v", err)
	}

	return c.Value, nil
}

// GetBool retrive a boolean value from the context using DataStore
func (context *Context) GetBool(key string) (bool, error) {
	data, err := context.dataStore.Get(key)
	if err != nil {
		return false, err
	}

	c := struct {
		Key   string `json:"key"`
		Value bool   `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		return false, fmt.Errorf("Failed to unmarshal data, error %v", err)
	}

	return c.Value, nil
}

// Del deletes a value from the context using DataStore
func (context *Context) Del(key string) error {
	return context.dataStore.Del(key)
}
