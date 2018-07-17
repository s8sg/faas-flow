package function

import (
	"fmt"
	"github.com/robertkrimen/otto"
)

func loadRuntime(body string) (*otto.Otto, error) {
	runtime := otto.New()
	// Load the plugin file into the runtime before we
	// return it for use
	if _, err := runtime.Run(body); err != nil {
		return nil, fmt.Errorf("Failed to load Modifier Runtime, error %v", err)
	}
	return runtime, nil
}

func jsexecute(body string, data string) (string, error) {
	runtime, err := loadRuntime(body)

	// If we don't have a runtime all requests are accepted
	if err != nil {
		return "", err
	}

	v, err := runtime.ToValue(data)
	if err != nil {
		return "", fmt.Errorf("Failed to set request body to Runtime, error %v", err)
	}

	// By convention we will require plugins have a set name
	result, err := runtime.Call("modify", nil, v)
	if err != nil {
		return "", fmt.Errorf("Failed to execute modify() on Runtime, error %v", err)
	}
	// If the js function did not return a string error out
	// because the modifier is invalid
	out, err := result.ToString()
	if err != nil {
		return "", fmt.Errorf("\"modify()\" must return a string. Got %s", err)
	}
	return out, nil
}
