package sdk

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

type Operation struct {
	// Operations
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

// CreateFunction Create a function with execution name
func CreateFunction(name string) *Operation {
	operation := &Operation{}
	operation.Function = name
	operation.Header = make(map[string]string)
	operation.Param = make(map[string][]string)
	return operation
}

// CreateModifier Create a modifier
func CreateModifier(mod Modifier) *Operation {
	operation := &Operation{}
	operation.Mod = mod
	return operation
}

// CreateCallback Create a callback
func CreateCallback(url string) *Operation {
	operation := &Operation{}
	operation.CallbackUrl = url
	operation.Header = make(map[string]string)
	operation.Param = make(map[string][]string)
	return operation
}

func (operation *Operation) Addheader(key string, value string) {
	lKey := strings.ToLower(key)
	operation.Header[lKey] = value
}

func (operation *Operation) Addparam(key string, value string) {
	array, ok := operation.Param[key]
	if !ok {
		operation.Param[key] = make([]string, 1)
		operation.Param[key][0] = value
	} else {
		operation.Param[key] = append(array, value)
	}
}

func (operation *Operation) AddFailureHandler(handler FuncErrorHandler) {
	operation.FailureHandler = handler
}

func (operation *Operation) AddResponseHandler(handler RespHandler) {
	operation.OnResphandler = handler
}

func (operation *Operation) AddRequestHandler(handler ReqHandler) {
	operation.Requesthandler = handler
}

func (operation *Operation) GetParams() map[string][]string {
	return operation.Param
}

func (operation *Operation) GetHeaders() map[string]string {
	return operation.Header
}
