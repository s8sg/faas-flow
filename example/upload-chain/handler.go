package function

import (
	"encoding/json"
	"fmt"
	"github.com/s8sg/faaschain"
)

type Dimention struct {
	X int
	Y int
}

type Face struct {
	Min Dimention
	Max Dimention
}

type FaceResult struct {
	Faces       []Face
	Bounds      Face
	ImageBase64 string
}

// Handle a serverless request to chian
func Define(chain *faaschain.Fchain) (err error) {

	// define chain
	chain.Apply("facedetect", nil, nil).
		ApplyModifier(func(data []byte) ([]byte, error) {
			context := faaschain.GetContext()
			result := FaceResult{}
			err := json.Unmarshal(data, &result)
			if err != nil {
				return nil, fmt.Errorf("Failed to decode facedetect result, error %v", err)
			}
			switch len(result.Faces) {
			case 0:
				return nil, fmt.Errorf("No face detected, picture should contain one face")
			case 1:
				return context.GetPhaseInput(), nil
			}
			return nil, fmt.Errorf("More than one face detected, picture should have single face")
		}).
		Apply("colorization", nil, nil).
		Apply("image-resizer", nil, nil)

	return nil
}
