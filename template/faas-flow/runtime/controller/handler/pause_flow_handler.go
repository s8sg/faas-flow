package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/faasflow/sdk/executor"
)

func PauseFlowHandler(w http.ResponseWriter, req *http.Request, id string, ex executor.Executor) ([]byte, error) {
	log.Printf("Pausing flow %s\n", id)

	flowExecutor := executor.CreateFlowExecutor(ex, nil)
	err := flowExecutor.Pause(id)
	if err != nil {
		return nil, fmt.Errorf("failed to pause request %s, check if request is active", id)
	}

	return []byte("Successfully paused request " + id), nil
}
