package lib

import (
	//	"encoding/json"
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
	chain := NewFaaschain("127.0.0.1:8080")
	if chain == nil {
		t.Errorf("Creating faas chain: got %v", chain)
		t.Fail()
	}
}

func TestApply(t *testing.T) {
	chain := NewFaaschain("127.0.0.1:8080")
	chain.Apply("compress", map[string]string{"Method": "Post"}, nil).Apply("upload", map[string]string{"Method": "Post"}, map[string][]string{"URL": []string{"my.file.storage/s8sg"}})
}

func TestApplyFunction(t *testing.T) {
	//chain := NewFaaschain("127.0.0.1:8080")

}

func TestApplyAsync(t *testing.T) {

}

func TestApplyAsyncFunction(t *testing.T) {

}

func TestBuild(t *testing.T) {

}

func TestInvoke(t *testing.T) {

}
