package lib

import (
	"strings"
)

type Function struct {
	Header   map[string]string   // The HTTP call header
	Param    map[string][]string // The Parameter in Query string
	Endpoint string              // Endpoint for function (Next Function within a Phase)
}

func CreateFunction(endpoint string) *Function {
	function := &Function{Endpoint: endpoint}
	function.Header = make(map[string]string)
	function.Param = make(map[string][]string)
	return function
}

func (function *Function) Addheader(key string, value string) {
	lKey = strings.ToLower(key)
	function.Header[lKey] = value
}

func (function *Function) Addparam(key string, value string) {
	array, ok := function.Param[key]
	if !ok {
		function.Param[key] = make([]string, 1)
		function.Param[key][0] = value
	} else {
		append(array, value)
	}
}
