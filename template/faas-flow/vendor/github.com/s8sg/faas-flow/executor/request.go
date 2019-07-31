package executor

import (
	"encoding/json"
	"fmt"
)

// Request defines the body of async forward request to faasflow
type Request struct {
	Sign  string `json: "sign"`  // request signature
	ID    string `json: "id"`    // request ID
	Query string `json: "query"` // query string

	ExecutionState string `json: "state"` // Execution State (execution position / execution vertex)

	Data []byte `json: "data"` // Partial execution data
	// (empty if intermediate_stoarge enabled

	ContextStore map[string]string `json: "store"` // Context State for default DataStore
	// (empty if external Store is used
}

const (
	// A signature of SHA265 equivalent of "github.com/s8sg/faasflow"
	SIGN = "D9D98C7EBAA7267BCC4F0280FC5BA4273F361B00D422074985A41AE1338F1B61"
)

func buildRequest(id string,
	state string,
	query string,
	data []byte,
	contextstate map[string]string,
	sign string) *Request {

	request := &Request{
		Sign:           sign,
		ID:             id,
		ExecutionState: state,
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

func (req *Request) getContextStore() map[string]string {
	return req.ContextStore
}

func (req *Request) getQuery() string {
	return req.Query
}
