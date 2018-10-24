package function

import (
	"encoding/json"
	"strings"
)

// Handle a serverless request
func Handle(req []byte) string {
	name := struct {
		Firstname string
		Lastname  string
	}{}
	err := json.Unmarshal(req, &name)
	if err != nil {
		return "error " + err.Error()
	}
	name.Firstname = strings.Title(name.Firstname)
	name.Lastname = strings.Title(name.Lastname)

	data, _ := json.Marshal(name)
	return string(data)
}
