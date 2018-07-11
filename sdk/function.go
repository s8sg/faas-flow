package sdk

import (
	"strings"
)

type Function struct {
	Function string              `json:"function"` // The name of the function
	Header   map[string]string   `json:"header"`   // The HTTP call header
	Param    map[string][]string `json:"param"`    // The Parameter in Query string
}

func CreateFunction(name string) *Function {
	function := &Function{}
	function.Function = name
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

func (function *Function) GetName() string {
	return function.Function
}

func (function *Function) GetParams() map[string][]string {
	return function.Param
}

func (function *Function) GetHeaders() map[string]string {
	return function.Header
}
