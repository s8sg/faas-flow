package function

import (
	"testing"
)

func TestEchoExecute(t *testing.T) {
	dataset := []string{"", "Test", "{ \"json\" : \"data\" }"}
	function := `
	function modify(data) {
    		return data;
	}
	`
	for _, data := range dataset {
		resp, err := jsexecute(function, data)
		if err != nil {
			t.Errorf("failed to execute %v", err)
			t.Fail()
		}
		if data != resp {
			t.Errorf("Expected resp '%s' got '%s'", data, resp)
		}
	}
}

func TestJsonExecute(t *testing.T) {
	data := "{ \"json\" : \"json data\" }"
	function := `
	function modify(data) {
		var obj = JSON.parse(data);
    		return obj.json;
	}
	`

	resp, err := jsexecute(function, data)
	if err != nil {
		t.Errorf("failed to execute %v", err)
		t.Fail()
	}
	if resp != "json data" {
		t.Errorf("Expected resp '%s' got '%s'", "json data", resp)
	}
}
