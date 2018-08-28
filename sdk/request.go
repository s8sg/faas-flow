package sdk

import (
	"encoding/json"
	"fmt"
)

type Request struct {
	Sign     string                 `json: "sign"`
	ID       string                 `json: "id"`
	Data     []byte                 `json: "data"`
	Chaindef string                 `json: "chain-def"`
	State    map[string]interface{} `json: "state"`
}

const (
	// A signature of SHA265 equivalent of "github.com/s8sg/faaschain"
	SIGN = "D9D98C7EBAA7267BCC4F0280FC5BA4273F361B00D422074985A41AE1338F1B61"
)

func BuildRequest(id string, chaindef string, data []byte, state map[string]interface{}) *Request {
	request := &Request{Sign: SIGN, ID: id, Chaindef: chaindef, Data: data, State: state}
	return request
}

func DecodeRequest(data []byte) (*Request, error) {
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

func (req *Request) Encode() ([]byte, error) {
	return json.Marshal(req)
}

func (req *Request) GetChain() (*Chain, error) {
	return DecodeChain([]byte(req.Chaindef))
}

func (req *Request) GetData() []byte {
	return req.Data
}

func (req *Request) GetID() string {
	return req.ID
}

func (req *Request) GetState() map[string]interface{} {
	return req.State
}
