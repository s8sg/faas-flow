package controller

import (
	"handler/runtime/controller/handler"
	"net/http"

	"handler/runtime"

	"github.com/julienschmidt/httprouter"
)

func router(runtime runtime.Runtime) http.Handler {
	router := httprouter.New()
	router.POST("/flow/execute", newRequestHandlerWrapper(runtime, handler.ExecuteFlowHandler))
	router.POST("/flow/:id/forward", newRequestHandlerWrapper(runtime, handler.PartialExecuteFlowHandler))
	router.POST("/flow/:id/pause", newRequestHandlerWrapper(runtime, handler.PauseFlowHandler))
	router.POST("/flow/:id/resume", newRequestHandlerWrapper(runtime, handler.ResumeFlowHandler))
	router.POST("/flow/:id/stop", newRequestHandlerWrapper(runtime, handler.StopFlowHandler))
	router.GET("/flow/:id/state", newRequestHandlerWrapper(runtime, handler.FlowStateHandler))
	router.POST("/", newRequestHandlerWrapper(runtime, handler.LegacyRequestHandler))
	router.GET("/", newRequestHandlerWrapper(runtime, handler.LegacyRequestHandler))
	return router
}
