package main

import (
	"encoding/json"
	"fmt"
)

const (
	EXECUTE = "EXECUTE"
	DEFINE  = "DEFINE"
	REMOVE  = "REMOVE"
)

type Function struct {
	Name   string            `json:"name"`   // Name of the function to execute
	Params map[string]string `json:"params"` // The params that will be sent a query param
}

type RequestBody struct {
	Raw []byte `json:"raw"`
}

type Request struct {
	Chain    string      `json:"chain"`    // name of the chain
	Type     string      `json:"type"`     // Flag that specify the request type
	Executes []Function  `json:"executes"` // the list of function that will be executed in the given order
	Body     RequestBody `json:"body"`     // the raw data
}

// Parse and returns Request
func ParseRequest(data []byte) (*Request, error) {
	request := &Request{}
	err := json.Unmarshal(data, request)
	if err != nil {
		return nil, err
	}
	return request, nil
}

// Get Request Dependency (list of function name)
func (req *Request) GetDependency() []string {
	list := make([]string)

	if req.Executes != nil {

		for _, function := range req.Executes {
			append(list, function.Name)
		}
	}
	return list
}
