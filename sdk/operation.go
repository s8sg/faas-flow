package sdk

import (
	"net/http"
	"strings"
)

// Serializer defintion for the data serilizer of function
type Serializer func(map[string][]byte) ([]byte, error)

// FuncErrorHandler the error handler for OnFailure() options
type FuncErrorHandler func(error) error

// Modifier definition for Modify() call
type Modifier func([]byte) ([]byte, error)

// RespHandler definition for OnResponse() option on function
type RespHandler func(*http.Response) ([]byte, error)

type Operation struct {
	// Operations
	Function    string   // The name of the function
	CallbackUrl string   // Callback Url
	Mod         Modifier // Modifier

	// Optional Options
	Header map[string]string   // The HTTP call header
	Param  map[string][]string // The Parameter in Query string

	serializer Serializer // The serializer serialize multiple input into one

	FailureHandler FuncErrorHandler // The Failure handler of the function
	OnResphandler  RespHandler      // The http Resp handler of the function
}

// Create a function with execution name
func CreateFunction(name string) *Operation {
	function := &Operation{}
	function.Function = name
	function.Header = make(map[string]string)
	function.Param = make(map[string][]string)
	return function
}

// Create a modifier (it is encapsulated as a function)
func CreateModifier(mod Modifier) *Operation {
	function := &Operation{}
	function.Mod = mod
	return function
}

// Create a callback (it is encapsulated as a function)
func CreateCallback(url string) *Operation {
	function := &Operation{}
	function.CallbackUrl = url
	function.Header = make(map[string]string)
	function.Param = make(map[string][]string)
	return function
}

func (function *Operation) Addheader(key string, value string) {
	lKey := strings.ToLower(key)
	function.Header[lKey] = value
}

func (function *Operation) Addparam(key string, value string) {
	array, ok := function.Param[key]
	if !ok {
		function.Param[key] = make([]string, 1)
		function.Param[key][0] = value
	} else {
		function.Param[key] = append(array, value)
	}
}

func (function *Operation) AddSerializer(serializer Serializer) {
	function.serializer = serializer
}

func (function *Operation) AddFailureHandler(handler FuncErrorHandler) {
	function.FailureHandler = handler
}

func (function *Operation) AddResponseHandler(handler RespHandler) {
	function.OnResphandler = handler
}

func (function *Operation) GetParams() map[string][]string {
	return function.Param
}

func (function *Operation) GetHeaders() map[string]string {
	return function.Header
}
