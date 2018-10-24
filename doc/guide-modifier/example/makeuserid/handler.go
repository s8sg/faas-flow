package function

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/s8sg/faasflow"
)

// Define provide definiton of the workflow
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
	flow.
		Modify(func(data []byte) ([]byte, error) {
			name := struct {
				Firstname string
				Lastname  string
			}{}
			err := json.Unmarshal(data, &name)
			if err != nil {
				return nil, err
			}
			if name.Firstname == "" || name.Lastname == "" {
				return nil, fmt.Errorf("Firstname and Lastname must be provided")
			}
			return data, nil
		}).
		Apply("titleize", faasflow.Sync).
		Modify(func(data []byte) ([]byte, error) {
			name := struct {
				Firstname string
				Lastname  string
			}{}
			err := json.Unmarshal(data, &name)
			if err != nil {
				return nil, err
			}
			user := struct {
				XMLName   xml.Name `xml:"user"`
				Firstname string   `xml:"Firstname"`
				Lastname  string   `xml:"Lastname"`
			}{}
			user.Firstname = name.Firstname
			user.Lastname = name.Lastname
			resp, _ := xml.Marshal(user)
			return resp, nil
		}).
		Apply("format", faasflow.Sync)
	return
}
