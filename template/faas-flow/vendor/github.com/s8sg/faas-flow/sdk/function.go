package sdk

import (
	"net/http"
	"strings"
)

// Serializer defintion for the data serilizer of function
type Serializer func(...[]byte) ([]byte, error)

// FuncErrorHandler the error handler for OnFailure() options
type FuncErrorHandler func(error) error

// Modifier definition for Modify() call
type Modifier func([]byte) ([]byte, error)

// RespHandler definition for OnResponse() option on function
type RespHandler func(*http.Response) ([]byte, error)

type Function struct {
	Function    string `json:"function"` // The name of the function
	CallbackUrl string `json:"callback"` // Callback Url

	Mod Modifier `json:"-"` // Modifier

	Header map[string]string   `json:"header"` // The HTTP call header
	Param  map[string][]string `json:"param"`  // The Parameter in Query string

	serializer Serializer `json:"-"`

	FailureHandler FuncErrorHandler `json:"-"` // The Failure handler of the function
	OnResphandler  RespHandler      `json:"-"` // The http Resp handler of the function
}

// Create a function with execution name
func CreateFunction(name string) *Function {
	function := &Function{}
	function.Function = name
	function.Header = make(map[string]string)
	function.Param = make(map[string][]string)
	return function
}

// Create a modifier (it is encapsulated as a function)
func CreateModifier(mod Modifier) *Function {
	function := &Function{}
	function.Mod = mod
	return function
}

// Create a callback (it is encapsulated as a function)
func CreateCallback(url string) *Function {
	function := &Function{}
	function.CallbackUrl = url
	function.Header = make(map[string]string)
	function.Param = make(map[string][]string)
	return function
}

func (function *Function) Addheader(key string, value string) {
	lKey := strings.ToLower(key)
	function.Header[lKey] = value
}

func (function *Function) Addparam(key string, value string) {
	array, ok := function.Param[key]
	if !ok {
		function.Param[key] = make([]string, 1)
		function.Param[key][0] = value
	} else {
		function.Param[key] = append(array, value)
	}
}

func (function *Function) AddSerializer(serializer Serializer) {
	function.serializer = serializer
}

func (function *Function) AddFailureHandler(handler FuncErrorHandler) {
	function.FailureHandler = handler
}

func (function *Function) AddResponseHandler(handler RespHandler) {
	function.OnResphandler = handler
}

func (function *Function) GetName() string {
	return function.Function
}

func (function *Function) GetParams() map[string][]string {
	return function.Param
}

func (function *Function) GetHeaders() map[string]string {
	return function.Header
}
