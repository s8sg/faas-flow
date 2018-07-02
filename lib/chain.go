package lib

import (
	"bytes"
	"context"
	"github.com/s8sg/faas-chain/sdk"
	"io"
	"io/ioutil"
	"path/filepath"
)

type Faaschain struct {
	chain    sdk.Chain
	url      string
	reader   io.Reader
	chainDef []byte
}

func NewFaaschain(faasurl string, reder io.Reader) *Faaschain {
	fchain := &Faaschain{}
	fchain.chain = &sdk.CreateChain()
	fchain.reader = reader
	fchain.url = path.Join(faasurl, "function/faas-chain")
}

func (fchain *Faaschain) ApplyFunction(function sdk.Function) *Faaschain {
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
	Function := CreateFunction(function)
	if header != nil {
		for key, value := range header {
			Function.Addheader(key, value)
		}
	}
	if param != nil {
		for key, array := range param {
			for _, value := range array {
				Function.Addparam(key, value)
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
	phase.AddFunction(Function)
	return fchain
}

func (fchain *Faaschain) ApplyFunctionAsync(function sdk.Function) *Faaschain {
	phase := sdk.CreateExecutionPhase()
	fchain.chain.AddPhase(phase)
	phase.AddFunction(function)
	return fchain
}

func (fchain *Faaschain) ApplyAsync(function string, header map[string]string, param map[string][]string) *Faaschain {
	Function := CreateFunction(function)
	if header != nil {
		for key, value := range header {
			Function.Addheader(key, value)
		}
	}
	if param != nil {
		for key, array := range param {
			for _, value := range array {
				Function.Addparam(key, value)
			}
		}
	}
	phase := sdk.CreateExecutionPhase()
	fchain.chain.AddPhase(phase)
	phase.AddFunction(function)
	return fchain
}

func (fchain *Faaschain) Build() error {
	fchain.chain.Data = ioutil.Readall(fchain.reader)
	fchain.chainDef, err = fchain.chain.Encode()
	if err != nil {
		return err
	}
}

func buildUpstreamRequest(url string, data []byte) *http.Request {

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	if data != nil {
		req.Body = bytes.NewReader(data)
	}

	return upstreamReq
}

func (fchain *Faaschain) Invoke(ctx context.Context) (io.Reader, error) {
	url := fchain.url
	client := &http.Client{}
	upstreamReq := buildUpstreamRequest(url, fchain.chain.Data)
	res, resErr := client.Do(upstreamReq.WithContext(ctx))
	if resErr != nil {
		return nil, resErr
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("status code: %d", res.StatusCode)
	}
	return res.Body, nil
}
