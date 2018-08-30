package main

import (
	"encoding/json"
	"fmt"
)

// Request defines the body of async forward request to faaschain
type Request struct {
	Sign           string `json: "sign"`  // request signature
	ID             string `json: "id"`    // request ID
	Query          string `json: "query"` // query string
	ExecutionState string `json: "state"` // Execution State
	Data           []byte `json: "data"`  // Partial execution data

	ContextState map[string]interface{} `json: "state"` // Context State for default StateManager
}

const (
	// A signature of SHA265 equivalent of "github.com/s8sg/faaschain"
	SIGN = "D9D98C7EBAA7267BCC4F0280FC5BA4273F361B00D422074985A41AE1338F1B61"
)

func buildRequest(id string,
	state string,
	query string,
	data []byte,
	contextstate map[string]interface{}) *Request {

	request := &Request{
		Sign:           SIGN,
		ID:             id,
		ExecutionState: state,
		Query:          query,
		Data:           data,
		ContextState:   contextstate,
	}
	return request
}

func decodeRequest(data []byte) (*Request, error) {
	request := &Request{}
	err := json.Unmarshal(data, request)
	if err != nil {
		return nil, err
	}
	if request.Sign != SIGN {
		return nil, fmt.Errorf("the request signature doesn't match")
	}
	return request, nil
}

func (req *Request) encode() ([]byte, error) {
	return json.Marshal(req)
}

func (req *Request) getData() []byte {
	return req.Data
}

func (req *Request) getID() string {
	return req.ID
}

func (req *Request) getExecutionState() string {
	return req.ExecutionState
}

func (req *Request) getContextState() map[string]interface{} {
	return req.ContextState
}

func (req *Request) getQuery() string {
	return req.Query
}
