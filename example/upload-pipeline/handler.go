package function

import (
	"encoding/json"
	"fmt"
	faasflow "github.com/s8sg/faas-flow"
	"log"
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

// Define the pipeline definition
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {

	// define Pipleline
	flow.
		Modify(func(data []byte) ([]byte, error) {
			context.Set("rawImage", data)
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
				data, err := context.GetBytes("rawImage")
				if err != nil {
					return nil, fmt.Errorf("Failed to retrive picture from state, error %v", err)
				}
				return data, nil
			}
			return nil, fmt.Errorf("More than one face detected, picture should contain single face")
		}).
		Apply("colorization", faasflow.Sync).
		Apply("image-resizer", faasflow.Sync).
		OnFailure(func(err error) ([]byte, error) {
			log.Printf("Failed to upload picture for request id %s, error %v",
				context.GetRequestId(), err)
			errdata := fmt.Sprintf("{\"error\": \"%s\"}", err.Error())

			return []byte(errdata), err
		})

	return nil
}

// DefineStateStore provides the override of the default StateStore
func DefineStateStore() (faasflow.StateStore, error) {
	return nil, nil
}

// ProvideDataStore provides the override of the default DataStore
func DefineDataStore() (faasflow.DataStore, error) {
	return nil, nil
}
