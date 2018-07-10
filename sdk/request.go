package sdk

import (
	"encoding/json"
)

type Request struct {
	Chaindef string `json: "definition"`
	Data     []byte `json: "data"`
}

func BuildRequest(chaindef string, data []byte) *Request {
	request := &Request{Chaindef: chaindef, Data: data}
	return request
}

func DecodeRequest(data []byte) (*Request, error) {
	request := &Request{}
	err := json.Unmarshal(data, request)
	if err != nil {
		return nil, err
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
