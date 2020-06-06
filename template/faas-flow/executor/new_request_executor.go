package executor

import (
	"net/http"
)

func newRequestExecutor(request *http.Request) (*openFaasExecutor, error) {
	ex := &openFaasExecutor{stateStore: stateStore, dataStore: dataStore}

	err := ex.init(request)
	if err != nil {
		return nil, err
	}

	return ex, nil
}
