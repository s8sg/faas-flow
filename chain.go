package lib

import (
	"context"
	"fmt"
	"github.com/s8sg/faas-chain/sdk"
	"io"
	"net/http"
	"path/filepath"
)

type Faaschain struct {
	chain    *sdk.Chain
	url      string
	chainDef []byte
}

func NewFaaschain(faasurl string) *Faaschain {
	fchain := &Faaschain{}
	fchain.chain = sdk.CreateChain()
	fchain.url = filepath.Join(faasurl, "function/faas-chain")
	return fchain
}

func (fchain *Faaschain) ApplyFunction(function *sdk.Function) *Faaschain {
	var phase *sdk.Phase
	if len(fchain.chain.Phases) == 0 {
		phase = sdk.CreateExecutionPhase()
		fchain.chain.AddPhase(phase)
	} else {
		phase = fchain.chain.GetLatestPhase()
	}
	phase.AddFunction(function)
	return fchain
}

func (fchain *Faaschain) Apply(function string, header map[string]string, param map[string][]string) *Faaschain {
	newfunc := sdk.CreateFunction(function)
	if header != nil {
		for key, value := range header {
			newfunc.Addheader(key, value)
		}
	}
	if param != nil {
		for key, array := range param {
			for _, value := range array {
				newfunc.Addparam(key, value)
			}
		}
	}
	var phase *sdk.Phase
	if len(fchain.chain.Phases) == 0 {
		phase = sdk.CreateExecutionPhase()
		fchain.chain.AddPhase(phase)
	} else {
		phase = fchain.chain.GetLatestPhase()
	}
	phase.AddFunction(newfunc)
	return fchain
}

func (fchain *Faaschain) ApplyFunctionAsync(function *sdk.Function) *Faaschain {
	phase := sdk.CreateExecutionPhase()
	fchain.chain.AddPhase(phase)
	phase.AddFunction(function)
	return fchain
}

func (fchain *Faaschain) ApplyAsync(function string, header map[string]string, param map[string][]string) *Faaschain {
	newfunc := sdk.CreateFunction(function)
	if header != nil {
		for key, value := range header {
			newfunc.Addheader(key, value)
		}
	}
	if param != nil {
		for key, array := range param {
			for _, value := range array {
				newfunc.Addparam(key, value)
			}
		}
	}
	phase := sdk.CreateExecutionPhase()
	fchain.chain.AddPhase(phase)
	phase.AddFunction(newfunc)
	return fchain
}

func buildUpstreamRequest(url string, reader io.ReadCloser) *http.Request {

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	if data != nil {
		req.Body = reader
	}

	return req
}

func (fchain *Faaschain) Invoke(ctx context.Context, reader io.ReadCloser) (io.ReadCloser, error) {

	url := fchain.url
	client := &http.Client{}
	upstreamReq := buildUpstreamRequest(url, reader)
	res, resErr := client.Do(upstreamReq.WithContext(ctx))
	if resErr != nil {
		return nil, resErr
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("status code: %d", res.StatusCode)
	}
	return res.Body, nil
}
