package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	fchain "github.com/s8sg/faaschain"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
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

func getQuery(key string) string {
	values, err := url.ParseQuery(os.Getenv("Http_Query"))
	if err != nil {
		return ""
	}
	return values.Get("file")

}

// Upload file upload logic
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

// Handle a serverless request to chain
func Define(chain *fchain.Fchain, context *fchain.Context) (err error) {

	// Define Chain
	chain.
		ApplyModifier(func(data []byte) ([]byte, error) {
			// Set the name of the file (error if not specified)
			filename := getQuery("file")
			if filename != "" {
				context.Set("file", filename)
			} else {
				return nil, fmt.Errorf("Provide file name with `--query file=<name>`")
			}
			// Set data to reuse after facedetect
			context.Set("raw", data)
			return data, nil
		}).
		Apply("facedetect").
		ApplyModifier(func(data []byte) ([]byte, error) {
			// validate face
			err := validateFace(data)
			if err != nil {
				return nil, err
			}
			// Get data from context
			rawdata, err := context.Get("raw")
			b, ok := rawdata.([]byte)
			if err != nil || !ok {
				return nil, fmt.Errorf("Failed to retrive picture from state, error %v %v", err, ok)
			}
			return b, err
		}).
		Apply("colorization").
		Apply("image-resizer").
		ApplyModifier(func(data []byte) ([]byte, error) {
			// get file name from context
			file, err := context.Get("file")
			filename, ok := file.(string)
			if err != nil || !ok {
				return nil, fmt.Errorf("Failed to get file name in context, %s %v", filename, err)
			}
			// upload file to storage
			err = upload(&http.Client{}, "http://gateway:8080/function/file-storage",
				filename, bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
			return nil, nil
		}).
		OnFailure(func(err error) {
			log.Printf("Failed to upload picture for request id %s, error %v",
				context.GetRequestId(), err)
		}).
		Finally(func(state string) {
			// Optional (cleanup)
			context.Del("raw")
			context.Del("file")
		})

	return nil
}
