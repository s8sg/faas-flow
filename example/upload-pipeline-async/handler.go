package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	faasflow "github.com/s8sg/faas-flow"
	minioDataStore "github.com/s8sg/faas-flow-minio-datastore"
	"io"
	"log"
	"mime/multipart"
	"net/http"
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

// Defines a Pipeline
func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {

	// Define Pipeline
	flow.
		Modify(func(data []byte) ([]byte, error) {
			// Set the name of the file (error if not specified)
			filename := context.Query.Get("file")
			if filename == "" {
				return nil, fmt.Errorf("Provide file name with `--query file=<name>`")
			}
			// Set data to reuse after facedetect
			err := context.Set("rawImage", data)
			if err != nil {
				return nil, fmt.Errorf("Failed to upload picture to state, error %v", err)
			}
			return data, nil
		}).
		Apply("facedetect").
		Modify(func(data []byte) ([]byte, error) {
			// validate face
			err := validateFace(data)
			if err != nil {
				filename := context.Query.Get("file")
				return nil, fmt.Errorf("File %s, %v", filename, err)
			}
			// Get data from context
			rawdata, err := context.GetBytes("rawImage")
			if err != nil {
				return nil, fmt.Errorf("Failed to retrive picture from state, error %v", err)
			}
			return rawdata, err
		}).
		Apply("colorization").
		Apply("image-resizer").
		Modify(func(data []byte) ([]byte, error) {
			// get file name from context
			filename := context.Query.Get("file")
			// upload file to storage
			err = upload(&http.Client{}, "http://gateway:8080/function/file-storage",
				filename, bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
			return nil, nil
		}).
		OnFailure(func(err error) ([]byte, error) {
			log.Printf("Failed to upload picture for request id %s, error %v",
				context.GetRequestId(), err)
			errdata := fmt.Sprintf("{\"error\": \"%s\"}", err.Error())

			return []byte(errdata), err
		}).
		Finally(func(state string) {
			// Optional (cleanup)
			// Cleanup is not needed if using default DataStore
			context.Del("rawImage")
		})

	return nil
}

// DefineStateStore provides the override of the default StateStore
func DefineStateStore() (faasflow.StateStore, error) {
	return nil, nil
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
