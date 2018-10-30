package function

import (
	"encoding/xml"
)

type User struct {
	XMLName   xml.Name `xml:"user"`
	Firstname string   `xml:"Firstname"`
	Lastname  string   `xml:"Lastname"`
}

// Handle a serverless request
func Handle(req []byte) string {
	user := &User{}
	err := xml.Unmarshal(req, user)
	if err != nil {
		return "error " + err.Error()
	}

	return user.Lastname + "." + user.Firstname
}
