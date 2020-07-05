package runtime

import (
	"github.com/faasflow/sdk/executor"
	"net/http"
)

type Runtime interface {
	Init() error
	CreateExecutor(*http.Request) (executor.Executor, error)
}
