package function

import (
	"encoding/json"
	"fmt"
	faasflow "github.com/s8sg/faasflow"
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
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {

	// define chain
	flow.
		Modify(func(data []byte) ([]byte, error) {
			context.Set("raw", data)
			return data, nil
		}).
		Apply("facedetect", faasflow.Sync).
		Modify(func(data []byte) ([]byte, error) {
			result := FaceResult{}
			err := json.Unmarshal(data, &result)
			if err != nil {
				return nil, fmt.Errorf("Failed to decode facedetect result, error %v", err)
			}
			switch len(result.Faces) {
			case 0:
				return nil, fmt.Errorf("No face detected, picture should contain one face")
			case 1:
				data, err := context.Get("raw")
				b, ok := data.([]byte)
				if err != nil || !ok {
					return nil, fmt.Errorf("Failed to retrive picture from state, error %v %v", err, ok)
				}

				return b, nil
			}
			return nil, fmt.Errorf("More than one face detected, picture should have single face")
		}).
		Apply("colorization", faasflow.Sync).
		Apply("image-resizer", faasflow.Sync)

	return nil
}
