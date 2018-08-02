package faaschain

import (
	//	"bytes"
	//	"context"
	//	"fmt"
	"github.com/s8sg/faaschain/sdk"
	//	"io"
	//	"io/ioutil"
	//	"net/http"
	"net/url"
	"path"
)

type Fchain struct {
	chain    *sdk.Chain
	id       string
	url      string
	asyncUrl string
	chainDef []byte
}

func NewFaaschain(faasurl string) *Fchain {
	fchain := &Fchain{}
	fchain.chain = sdk.CreateChain()
	u, _ := url.Parse(faasurl)
	u.Path = path.Join(u.Path, "function/faaschain")
	fchain.url = u.String()
	u, _ = url.Parse(faasurl)
	u.Path = path.Join(u.Path, "async-function/faaschain")
	fchain.asyncUrl = u.String()
	return fchain
}

func (fchain *Fchain) SetId(id string) {
	fchain.id = id
}

func (fchain *Fchain) ApplyModifier(mod sdk.Modifier) *Fchain {
	var phase *sdk.Phase
	newfunc := sdk.CreateModifier(mod)
	if len(fchain.chain.Phases) == 0 {
		phase = sdk.CreateExecutionPhase()
		fchain.chain.AddPhase(phase)
	} else {
		phase = fchain.chain.GetLastPhase()
	}
	phase.AddFunction(newfunc)
	return fchain
}

func (fchain *Fchain) ApplyFunction(function *sdk.Function) *Fchain {
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

func (fchain *Fchain) Apply(function string, header map[string]string, param map[string][]string) *Fchain {
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

func (fchain *Fchain) ApplyFunctionAsync(function *sdk.Function) *Fchain {
	phase := sdk.CreateExecutionPhase()
	fchain.chain.AddPhase(phase)
	phase.AddFunction(function)
	return fchain
}

func (fchain *Fchain) ApplyAsync(function string, header map[string]string, param map[string][]string) *Fchain {
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

func (fchain *Fchain) Build() (err error) {
	fchain.chainDef, err = fchain.chain.Encode()
	return err
}

func (fchain *Fchain) GetDefinition() string {
	return string(fchain.chainDef)
}

func (fchain *Fchain) GetChain() *sdk.Chain {
	return fchain.chain
}

func (fchain *Fchain) GetId() string {
	return fchain.id
}

func (fchain *Fchain) GetUrl() string {
	return fchain.url
}

func (fchain *Fchain) GetAsyncUrl() string {
	return fchain.asyncUrl
}
