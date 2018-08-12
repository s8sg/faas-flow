package faaschain

import (
	"github.com/s8sg/faaschain/sdk"
	"testing"
)

var (
	data = []byte("<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 100 100\">" +
		"<path d=\"M30,1h40l29,29v40l-29,29h-40l-29-29v-40z\" stroke=\"#000\" fill=\"none\"/> " +
		"<path d=\"M31,3h38l28,28v38l-28,28h-38l-28-28v-38z\" fill=\"#a23\"/> " +
		"<text x=\"50\" y=\"68\" font-size=\"48\" fill=\"#FFF\" text-anchor=\"middle\"> " +
		"<![CDATA[410]]> " +
		"</text></svg>")
)

func TestChainCreate(t *testing.T) {
	chain := NewFaaschain("http://127.0.0.1:8080")
	if chain == nil {
		t.Errorf("Creating faas chain: got %v", chain)
		t.Fail()
	}
}

func TestApply(t *testing.T) {
	chain := NewFaaschain("http://127.0.0.1:8080")
	chain.Apply("compress", map[string]string{"Method": "Post"}, nil).Apply("upload", map[string]string{"Method": "Post"}, map[string][]string{"URL": {"my.file.storage/s8sg"}})
}

func TestApplyFunction(t *testing.T) {
	chain := NewFaaschain("http://127.0.0.1:8080")
	func1 := sdk.CreateFunction("compress")
	func1.Addheader("Method", "Post")
	func2 := sdk.CreateFunction("upload")
	func2.Addheader("Method", "Post")
	func2.Addparam("URL", "my.file.storage/s8sg")
	chain.ApplyFunction(func1).ApplyFunction(func2)
}

func TestApplyAsync(t *testing.T) {
	chain := NewFaaschain("http://127.0.0.1:8080")
	chain.ApplyAsync("compress", map[string]string{"Method": "Post"}, nil).ApplyAsync("upload", map[string]string{"Method": "Post"}, map[string][]string{"URL": {"my.file.storage/s8sg"}})
}

func TestApplyAsyncFunction(t *testing.T) {
	chain := NewFaaschain("http://127.0.0.1:8080")
	func1 := sdk.CreateFunction("compress")
	func1.Addheader("Method", "Post")
	func2 := sdk.CreateFunction("upload")
	func2.Addheader("Method", "Post")
	func2.Addparam("URL", "my.file.storage/s8sg")
	chain.ApplyFunctionAsync(func1).ApplyFunctionAsync(func2)
}

func TestBuild(t *testing.T) {
	chain1 := NewFaaschain("http://127.0.0.1:8080")
	upload := sdk.CreateFunction("upload")
	upload.Addheader("Method", "Post")
	upload.Addparam("URL", "my.file.storage/s8sg")
	chain1.Apply("compress", map[string]string{"Method": "Post"}, nil).ApplyFunction(upload)
	err := chain1.Build()
	if err != nil {
		t.Errorf("Failled to build chain, got error %v", err)
		t.Fail()
	}
	def := chain1.GetDefinition()
	if def == "" {
		t.Errorf("Failled to build chain, got empty %v", err)
		t.Fail()
	}
}
