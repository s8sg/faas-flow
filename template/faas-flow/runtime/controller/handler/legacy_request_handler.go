package handler

import (
	"handler/runtime/controller/util"
	"net/http"

	"github.com/faasflow/sdk/executor"
)

func LegacyRequestHandler(w http.ResponseWriter, r *http.Request, id string, ex executor.Executor) ([]byte, error) {
	var handler func(http.ResponseWriter, *http.Request, string, executor.Executor) ([]byte, error)

	switch {
	case util.IsDagExportRequest(r.URL.RawQuery):
		handler = GetDagHandler

	case util.GetPauseRequestID(r.URL.RawQuery) != "":
		id = util.GetPauseRequestID(r.URL.RawQuery)
		handler = PauseFlowHandler

	case util.GetStopRequestID(r.URL.RawQuery) != "":
		id = util.GetStopRequestID(r.URL.RawQuery)
		handler = StopFlowHandler

	case util.GetResumeRequestID(r.URL.RawQuery) != "":
		id = util.GetResumeRequestID(r.URL.RawQuery)
		handler = ResumeFlowHandler

	case util.GetStateRequestID(r.URL.RawQuery) != "":
		id = util.GetStateRequestID(r.URL.RawQuery)
		handler = FlowStateHandler

	default:
		id = r.Header.Get(util.RequestIdHeader)
		if id == "" {
			handler = ExecuteFlowHandler
		} else {
			handler = PartialExecuteFlowHandler
		}
	}

	body, err := handler(w, r, id, ex)
	return body, err
}
