package faasflow

import (
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
		id = "calback-" + operation.CallbackUrl[len(operation.CallbackUrl)-8:]
	}
	return id
}

func (operation *FaasOperation) GetProperties() map[string]string {

	result := make(map[string]string)

	result["isMod"] = "false"
	result["isFunction"] = "false"
	result["isCallback"] = "false"
	result["hasFailureHandler"] = "false"
	result["hasResponseHandler"] = "false"

	if operation.Mod != nil {
		result["isMod"] = "true"
	}
	if operation.Function != "" {
		result["isFunction"] = "true"
	}
	if operation.CallbackUrl != "" {
		result["isCallback"] = "true"
	}
	if operation.FailureHandler != nil {
		result["hasFailureHandler"] = "true"
	}
	if operation.OnResphandler != nil {
		result["hasResponseHandler"] = "true"
	}

	return result
}
