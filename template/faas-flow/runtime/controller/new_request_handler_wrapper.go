package controller

import (
	"fmt"
	"net/http"

	"handler/runtime"

	"github.com/faasflow/sdk/executor"
	"github.com/julienschmidt/httprouter"
)

func newRequestHandlerWrapper(runtime runtime.Runtime, handler func(http.ResponseWriter, *http.Request, string, executor.Executor) ([]byte, error)) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		id := params.ByName("id")

		ex, err := runtime.CreateExecutor(req)
		if err != nil {
			handleError(w, fmt.Sprintf("failed to execute request "+id))
			return
		}

		body, err := handler(w, req, id, ex)
		if err != nil {
			handleError(w, fmt.Sprintf("request failed to be processed. "+err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}
}
