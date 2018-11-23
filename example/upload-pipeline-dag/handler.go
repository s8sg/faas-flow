package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	faasflow "github.com/s8sg/faas-flow"
	consulStateStore "github.com/s8sg/faas-flow-consul-statestore"
	minioDataStore "github.com/s8sg/faas-flow-minio-datastore"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
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

// Upload file to the result storage
func upload(client *http.Client, url string, filename string, r io.Reader) (err error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	var fw io.Writer

	if x, ok := r.(io.Closer); ok {
		defer x.Close()
	}
	// Add an image file
	if fw, err = w.CreateFormFile("file", filename); err != nil {
		return
	}
	if _, err = io.Copy(fw, r); err != nil {
		return err
	}

	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Submit the request
	res, err := client.Do(req)
	if err != nil {
		return
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}
	return
}

// validateFace validate the no of face
func validateFace(data []byte) error {
	result := FaceResult{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return fmt.Errorf("Failed to decode facedetect result, error %v", err)
	}
	switch len(result.Faces) {
	case 0:
		return fmt.Errorf("No face detected, picture should contain one face")
	case 1:
		return nil
	}
	return fmt.Errorf("More than one face detected, picture should have single face")
}

// Define provide definiton of the workflow
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {

	// Create the dag
	uploadDag := faasflow.CreateDag()

	// Create a modifier vertex to validate request query
	uploadDag.CreateModifierVertex("validate-query", func(data []byte) ([]byte, error) {
		// Set the name of the file (error if not specified)
		filename := context.Query.Get("file")
		if filename == "" {
			return nil, fmt.Errorf("Provide file name with `--query file=<name>`")
		}
		return data, nil
	})

	// Create a function vertex to detect face
	uploadDag.CreateFunctionVertex("detect-face", "facedetect")

	// Create a function vertex to colorize image
	uploadDag.CreateFunctionVertex("colorize", "colorization")

	// Create a function vertex to compress image
	uploadDag.CreateFunctionVertex("compress", "image-resizer")

	// Create a modifier vertex to validate image and upload
	// It uses
	uploadDag.CreateModifierVertex("validate-and-upload", func(data []byte) ([]byte, error) {
		// get file name from context
		filename := context.Query.Get("file")
		// upload file to storage
		err = upload(&http.Client{}, "http://gateway:8080/function/file-storage",
			filename, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return nil, nil
	}, faasflow.Serializer(func(inputs map[string][]byte) ([]byte, error) {
		// Get facedetect result from input
		faceDetectResult := inputs["detect-face"]

		err := validateFace(faceDetectResult)
		if err != nil {
			filename := context.Query.Get("file")
			return nil, fmt.Errorf("File %s, %v", filename, err)
		}
		// Get converted image from input
		data := inputs["compress"]
		return data, nil
	}))

	// validate-query -> detect-face -> validate-upload
	uploadDag.AddEdge("validate-query", "detect-face")
	uploadDag.AddEdge("detect-face", "validate-and-upload")

	// validate-query -> colorize -> compress -> validate-and-upload
	uploadDag.AddEdge("validate-query", "colorize")
	uploadDag.AddEdge("colorize", "compress")
	uploadDag.AddEdge("compress", "validate-and-upload")

	// add the dag to the flow
	flow.ExecuteDag(uploadDag)

	flow.
		OnFailure(func(err error) ([]byte, error) {
			log.Printf("Failed to upload picture for request id %s, error %v",
				context.GetRequestId(), err)
			errdata := fmt.Sprintf("{\"error\": \"%s\"}", err.Error())

			return []byte(errdata), err
		}).
		Finally(func(state string) {
			// Cleanup is not needed if using default DataStore
		})

	return
}

// DefineStateStore provides the override of the default StateStore
func DefineStateStore() (faasflow.StateStore, error) {
	consulss, err := consulStateStore.GetConsulStateStore(os.Getenv("consul_url"), os.Getenv("consul_dc"))
	if err != nil {
		return nil, err
	}
	return consulss, nil
}

// ProvideDataStore provides the override of the default DataStore
func DefineDataStore() (faasflow.DataStore, error) {
	// initialize minio DataStore
	miniods, err := minioDataStore.InitFromEnv()
	if err != nil {
		return nil, err
	}

	return miniods, nil
}
