package function

import (
	"fmt"
	"github.com/s8sg/faaschain"
)

// Handle a serverless request to chian
func Define(chain *faaschain.Fchain, context *faaschain.Context) (err error) {
	chain.ApplyModifier(func(data []byte) ([]byte, error) {
		return []byte(fmt.Sprintf("you said \"%s\"", string(data))), nil
	})
	return
}
