package controller

import (
	"net/http"

	"github.com/faasflow/sdk/executor"
	"github.com/julienschmidt/httprouter"
)

func legacyRequestHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var handler func(http.ResponseWriter, *http.Request, string, executor.Executor) ([]byte, error)
	id := ""

	switch {
	case isDagExportRequest(r.URL.RawQuery):
		handler = getDagHandler

	case getPauseRequestID(r.URL.RawQuery) != "":
		id = getPauseRequestID(r.URL.RawQuery)
		handler = pauseFlowHandler

	case getStopRequestID(r.URL.RawQuery) != "":
		id = getStopRequestID(r.URL.RawQuery)
		handler = stopFlowHandler

	case getResumeRequestID(r.URL.RawQuery) != "":
		id = getResumeRequestID(r.URL.RawQuery)
		handler = resumeFlowHandler

	case getStateRequestID(r.URL.RawQuery) != "":
		id = getStateRequestID(r.URL.RawQuery)
		handler = flowStateHandler

	default:
		id = r.Header.Get("X-Faas-Flow-Reqid")
		handler = executeFlowHandler
	}

	p = append(p, httprouter.Param{
		Key:   "id",
		Value: id,
	})

	newRequestHandlerWrapper(handler)(w, r, p)
}
