package function

import (
	"github.com/s8sg/faaschain"
)

// Handle a serverless request to chian
func Define(chain *faaschain.Fchain) (err error) {

	// Define Chain
	chain.Apply("colorization", map[string]string{"method": "post"}, nil).
		Apply("image-resizer", map[string]string{"method": "post"}, nil)

	return nil
}
