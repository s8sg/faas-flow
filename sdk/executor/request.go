package executor

import (
	"encoding/json"
)

// Request defines the body of async forward request to faasflow
type Request struct {
	Sign        string `json: "sign"`         // request signature
	ID          string `json: "id"`           // request ID
	Query       string `json: "query"`        // query string
	CallbackUrl string `json: "callback-url"` // callback url

	ExecutionState string `json: "state"` // Execution State (execution position / execution vertex)

	Data []byte `json: "data"` // Partial execution data
	// (empty if intermediate_storage enabled

	ContextStore map[string]string `json: "store"` // Context State for default DataStore
	// (empty if external Store is used
}

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
