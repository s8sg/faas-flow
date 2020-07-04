package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/s8sg/faas-flow/sdk/executor"
)

func flowStateHandler(w http.ResponseWriter, req *http.Request, id string, ex executor.Executor) ([]byte, error) {
	log.Printf("Get flow state: %s\n", id)

	flowExecutor := executor.CreateFlowExecutor(ex, nil)
	state, err := flowExecutor.GetState(id)
	if err != nil {
		log.Printf(err.Error())
		return nil, fmt.Errorf("failed to get request state for %s, check if request is active", id)
	}

	return []byte(state), nil
}
