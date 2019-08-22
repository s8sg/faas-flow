package faasflow

import (
	"encoding/json"
	"net/http"
	"strings"
)

var (
	BLANK_MODIFIER = func(data []byte) ([]byte, error) { return data, nil }
)

// FuncErrorHandler the error handler for OnFailure() options
type FuncErrorHandler func(error) error

// Modifier definition for Modify() call
type Modifier func([]byte) ([]byte, error)

// RespHandler definition for OnResponse() option on operation
type RespHandler func(*http.Response) ([]byte, error)

// Reqhandler definition for RequestHdlr() option on operation
type ReqHandler func(*http.Request)

type FaasOperation struct {
	// FaasOperations
	Function    string   // The name of the function
	CallbackUrl string   // Callback Url
	Mod         Modifier // Modifier

	// Optional Options
	Header map[string]string   // The HTTP call header
	Param  map[string][]string // The Parameter in Query string

	FailureHandler FuncErrorHandler // The Failure handler of the operation
	Requesthandler ReqHandler       // The http request handler of the operation
	OnResphandler  RespHandler      // The http Resp handler of the operation
}

// createFunction Create a function with execution name
func createFunction(name string) *FaasOperation {
	operation := &FaasOperation{}
	operation.Function = name
	operation.Header = make(map[string]string)
	operation.Param = make(map[string][]string)
	return operation
}

// createModifier Create a modifier
func createModifier(mod Modifier) *FaasOperation {
	operation := &FaasOperation{}
	operation.Mod = mod
	return operation
}

// createCallback Create a callback
func createCallback(url string) *FaasOperation {
	operation := &FaasOperation{}
	operation.CallbackUrl = url
	operation.Header = make(map[string]string)
	operation.Param = make(map[string][]string)
	return operation
}

func (operation *FaasOperation) addheader(key string, value string) {
	lKey := strings.ToLower(key)
	operation.Header[lKey] = value
}

func (operation *FaasOperation) addparam(key string, value string) {
	array, ok := operation.Param[key]
	if !ok {
		operation.Param[key] = make([]string, 1)
		operation.Param[key][0] = value
	} else {
		operation.Param[key] = append(array, value)
	}
}

func (operation *FaasOperation) addFailureHandler(handler FuncErrorHandler) {
	operation.FailureHandler = handler
}

func (operation *FaasOperation) addResponseHandler(handler RespHandler) {
	operation.OnResphandler = handler
}

func (operation *FaasOperation) addRequestHandler(handler ReqHandler) {
	operation.Requesthandler = handler
}

func (operation *FaasOperation) GetParams() map[string][]string {
	return operation.Param
}

func (operation *FaasOperation) GetHeaders() map[string]string {
	return operation.Header
}

func (operation *FaasOperation) GetId() string {
	id := "modifier"
	switch {
	case operation.Function != "":
		id = operation.Function
	case operation.CallbackUrl != "":
		id = "callback-" + operation.CallbackUrl[len(operation.CallbackUrl)-12:]
	}
	return id
}

func (operation *FaasOperation) Encode() ([]byte, error) {
	return json.Marshal(operation)
}

func (operation *FaasOperation) Decode(data []byte) error {
	return json.Unmarshal(data, operation)
}

func (operation *FaasOperation) Execute(data []byte) ([]byte, error) {
	return nil, nil
}

func (operation *FaasOperation) GetProperties() map[string][]string {

	result := make(map[string][]string)

	isMod := "false"
	isFunction := "false"
	isCallback := "false"
	hasFailureHandler := "false"
	hasResponseHandler := "false"

	if operation.Mod != nil {
		isMod = "true"
	}
	if operation.Function != "" {
		isFunction = "true"
	}
	if operation.CallbackUrl != "" {
		isCallback = "true"
	}
	if operation.FailureHandler != nil {
		hasFailureHandler = "true"
	}
	if operation.OnResphandler != nil {
		hasResponseHandler = "true"
	}

	result["isMod"] = []string{isMod}
	result["isFunction"] = []string{isFunction}
	result["isCallback"] = []string{isCallback}
	result["hasFailureHandler"] = []string{hasFailureHandler}
	result["hasResponseHandler"] = []string{hasResponseHandler}

	return result
}

// Faasflow faas operations

// Modify adds a new modifier to the given vertex
func (node *Node) Modify(mod Modifier) *Node {
	newMod := createModifier(mod)
	node.unode.AddOperation(newMod)
	return node
}

// Apply adds a new function to the given vertex
func (node *Node) Apply(function string, opts ...Option) *Node {

	newfunc := createFunction(function)

	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if len(o.header) != 0 {
			for key, value := range o.header {
				newfunc.addheader(key, value)
			}
		}
		if len(o.query) != 0 {
			for key, array := range o.query {
				for _, value := range array {
					newfunc.addparam(key, value)
				}
			}
		}
		if o.failureHandler != nil {
			newfunc.addFailureHandler(o.failureHandler)
		}
		if o.responseHandler != nil {
			newfunc.addResponseHandler(o.responseHandler)
		}
		if o.requestHandler != nil {
			newfunc.addRequestHandler(o.requestHandler)
		}
	}

	node.unode.AddOperation(newfunc)
	return node
}

// Callback adds a new callback to the given vertex
func (node *Node) Callback(url string, opts ...Option) *Node {
	newCallback := createCallback(url)

	o := &Options{}
	for _, opt := range opts {
		o.reset()
		opt(o)
		if len(o.header) != 0 {
			for key, value := range o.header {
				newCallback.addheader(key, value)
			}
		}
		if len(o.query) != 0 {
			for key, array := range o.query {
				for _, value := range array {
					newCallback.addparam(key, value)
				}
			}
		}
		if o.failureHandler != nil {
			newCallback.addFailureHandler(o.failureHandler)
		}
		if o.responseHandler != nil {
			newCallback.addResponseHandler(o.responseHandler)
		}
		if o.requestHandler != nil {
			newCallback.addRequestHandler(o.requestHandler)
		}
	}

	node.unode.AddOperation(newCallback)
	return node
}
