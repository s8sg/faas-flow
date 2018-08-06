package function

import (
	"github.com/s8sg/faaschain"
)

// Handle a serverless request to chian
func Define(chain *faaschain.Fchain) (err error) {

	chain.Apply("colorization", map[string]string{"method": "post"}, nil).
		ApplyAsync("facedetect", map[string]string{"method": "post"}, map[string][]string{"output": []string{"image"}}).
		ApplyAsync("image-resizer", map[string]string{"method": "post"}, nil)

	return nil
}
