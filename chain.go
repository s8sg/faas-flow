package faaschain

import (
	"bytes"
	"context"
	"fmt"
	"github.com/s8sg/faaschain/sdk"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

type Faaschain struct {
	chain    *sdk.Chain
	url      string
	chainDef []byte
}

func NewFaaschain(faasurl string) *Faaschain {
	fchain := &Faaschain{}
	fchain.chain = sdk.CreateChain()
	u, _ := url.Parse(faasurl)
	u.Path = path.Join(u.Path, "function/faaschain")
	fchain.url = u.String()
	return fchain
}

func (fchain *Faaschain) ApplyFunction(function *sdk.Function) *Faaschain {
	var phase *sdk.Phase
	if len(fchain.chain.Phases) == 0 {
		phase = sdk.CreateExecutionPhase()
		fchain.chain.AddPhase(phase)
	} else {
		phase = fchain.chain.GetLastPhase()
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
		phase = fchain.chain.GetLastPhase()
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

func (fchain *Faaschain) Build() (err error) {
	fchain.chainDef, err = fchain.chain.Encode()
	return err
}

func (fchain *Faaschain) GetDefinition() string {
	return string(fchain.chainDef)
}

func buildUpstreamRequest(url string, chaindef string, data []byte) (*http.Request, error) {
	request := sdk.BuildRequest(chaindef, data)

	data, err := request.Encode()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

func (fchain *Faaschain) Invoke(ctx context.Context, reader io.Reader) (io.ReadCloser, error) {
	var res *http.Response
	var resErr error

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	url := fchain.url
	client := &http.Client{}
	upstreamReq, err := buildUpstreamRequest(url, string(fchain.chainDef), data)
	if err != nil {
		return nil, err
	}

	if ctx != nil {
		res, resErr = client.Do(upstreamReq.WithContext(ctx))
	} else {
		res, resErr = client.Do(upstreamReq)
	}
	if resErr != nil {
		return nil, resErr
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("status code: %d", res.StatusCode)
	}
	return res.Body, nil
}
