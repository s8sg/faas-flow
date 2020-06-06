package executor

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func router() http.Handler {
	router := httprouter.New()
	// router.POST("/flow/execute", newRequestHandlerWrapper(executeFlowHandler))
	router.POST("/flow/:id/execute", newRequestHandlerWrapper(executeFlowHandler))
	router.POST("/flow/:id/pause", newRequestHandlerWrapper(pauseFlowHandler))
	router.POST("/flow/:id/resume", newRequestHandlerWrapper(resumeFlowHandler))
	router.POST("/flow/:id/stop", newRequestHandlerWrapper(stopFlowHandler))
	router.GET("/flow/:id/state", newRequestHandlerWrapper(flowStateHandler))
	router.POST("/", legacyRequestHandler)
	router.GET("/", legacyRequestHandler)
	return router
}
