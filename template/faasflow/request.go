package main

import (
	"encoding/json"
	"fmt"
)

// Request defines the body of async forward request to faasflow
type Request struct {
	Sign           string `json: "sign"`      // request signature
	ID             string `json: "id"`        // request ID
	Query          string `json: "query"`     // query string
	ExecutionState string `json: "e-state"`   // Execution State
	DagVertex      string `json: "dag-state"` // Dag Execution State
	Data           []byte `json: "data"`      // Partial execution data

	ContextStore map[string]string `json: "state"` // Context State for default StateManager
}

const (
	// A signature of SHA265 equivalent of "github.com/s8sg/faasflow"
	SIGN = "D9D98C7EBAA7267BCC4F0280FC5BA4273F361B00D422074985A41AE1338F1B61"
)

func buildRequest(id string,
	state string,
	dagstate string,
	query string,
	data []byte,
	contextstate map[string]string) *Request {

	request := &Request{
		Sign:           SIGN,
		ID:             id,
		ExecutionState: state,
		DagVertex:      dagstate,
		Query:          query,
		Data:           data,
		ContextStore:   contextstate,
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

func (req *Request) getDagState() string {
	return req.DagVertex
}

func (req *Request) getContextStore() map[string]string {
	return req.ContextStore
}

func (req *Request) getQuery() string {
	return req.Query
}
