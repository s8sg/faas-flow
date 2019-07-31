package sdk

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// Context execution context and execution state
type Context struct {
	requestId string     // the request id
	node      string     // the execution position
	dataStore DataStore  // underline DataStore
	Query     url.Values // provides request Query
	State     string     // state of the request
	Name      string     // name of the faas-flow

	NodeInput map[string][]byte // stores inputs form each node
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
	// Set a value (override existing, or create one)
	Set(key string, value string) error
	// Get a value
	Get(key string) (string, error)
	// Compare and Update a value
	Update(key string, oldValue string, newValue string) error
	// Cleanup all the resorces in StateStore (called only once in a request span)
	Cleanup() error
}

// EventHandler handle flow events
type EventHandler interface {
	// Configure the EventHandler with flow name and request ID
	Configure(flowName string, requestId string)
	// Initialize an EventHandler (called only once in a request span)
	Init() error
	// ReportRequestStart report a start of request
	ReportRequestStart(requestId string)
	// ReportRequestEnd reports an end of request
	ReportRequestEnd(requestId string)
	// ReportRequestFailure reports a failure of a request with error
	ReportRequestFailure(requestId string, err error)
	// ReportExecutionForward report that an execution is forwarded
	ReportExecutionForward(nodeId string, requestId string)
	// ReportExecutionContinuation report that an execution is being continued
	ReportExecutionContinuation(requestId string)
	// ReportNodeStart report a start of a Node execution
	ReportNodeStart(nodeId string, requestId string)
	// ReportNodeStart report an end of a node execution
	ReportNodeEnd(nodeId string, requestId string)
	// ReportNodeFailure report a Node execution failure with error
	ReportNodeFailure(nodeId string, requestId string, err error)
	// ReportOperationStart reports start of an operation
	ReportOperationStart(operationId string, nodeId string, requestId string)
	// ReportOperationEnd reports an end of an operation
	ReportOperationEnd(operationId string, nodeId string, requestId string)
	// ReportOperationFailure reports failure of an operation with error
	ReportOperationFailure(operationId string, nodeId string, requestId string, err error)
	// Flush flush the reports
	Flush()
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
func CreateContext(id string, node string, name string,
	dstore DataStore) *Context {

	context := &Context{}
	context.requestId = id
	context.node = node
	context.Name = name
	context.State = StateOngoing
	context.dataStore = dstore
	context.NodeInput = make(map[string][]byte)

	return context
}

// GetRequestId returns the request id
func (context *Context) GetRequestId() string {
	return context.requestId
}

// GetPhase return the node no
func (context *Context) GetNode() string {
	return context.node
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
func (context *Context) GetInt(key string) int {
	data, err := context.dataStore.Get(key)
	if err != nil {
		panic(fmt.Sprintf("error %v", err))
	}

	c := struct {
		Key   string `json:"key"`
		Value int    `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal data, error %v", err))
	}

	return c.Value
}

// GetString retrive a string value from the context using DataStore
func (context *Context) GetString(key string) string {
	data, err := context.dataStore.Get(key)
	if err != nil {
		panic(fmt.Sprintf("error %v", err))
	}

	c := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal data, error %v", err))
	}

	return c.Value
}

// GetBytes retrive a byte array from the context using DataStore
func (context *Context) GetBytes(key string) []byte {
	data, err := context.dataStore.Get(key)
	if err != nil {
		panic(fmt.Sprintf("error %v", err))
	}

	c := struct {
		Key   string `json:"key"`
		Value []byte `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal data, error %v", err))
	}

	return c.Value
}

// GetBool retrive a boolean value from the context using DataStore
func (context *Context) GetBool(key string) bool {
	data, err := context.dataStore.Get(key)
	if err != nil {
		panic(fmt.Sprintf("error %v", err))
	}

	c := struct {
		Key   string `json:"key"`
		Value bool   `json:"value"`
	}{}
	err = json.Unmarshal([]byte(data), &c)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal data, error %v", err))
	}

	return c.Value
}

// Del deletes a value from the context using DataStore
func (context *Context) Del(key string) error {
	return context.dataStore.Del(key)
}
