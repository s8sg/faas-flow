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

// Upload file upload logic
func Upload(client *http.Client, url string, filename string, r io.Reader) (err error) {
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

// Handle a serverless request to chain
func Define(chain *fchain.Fchain, context *fchain.Context) (err error) {

	// Define Chain
	chain.
		ApplyModifier(func(data []byte) ([]byte, error) {
			bytes := "12eeee 24   21312312 ddd2d 31e 1 111ddddd23d2 21e 12"
			context.Set("raw", []byte(bytes))
			log.Println(bytes)
			log.Println(len(data))
			return data, nil
		}).
		Apply("facedetect").
		ApplyModifier(func(data []byte) ([]byte, error) {
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
				log.Println(len(data))
				log.Println(data)
				b, ok := data.(string)
				if err != nil || !ok {
					return nil, fmt.Errorf("Failed to retrive picture from state, error %v %v", err, ok)
				}
				log.Println(len(b))
				log.Println(b)
				return []byte(b), nil
			}
			return nil, fmt.Errorf("More than one face detected, picture should have single face")
		}).
		Apply("colorization").
		Apply("image-resizer").
		ApplyModifier(func(data []byte) ([]byte, error) {
			err = Upload(&http.Client{}, "http://gateway:8080/function/file-storage",
				"chris.jpg", bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
			return nil, nil
		})

	return nil
}
