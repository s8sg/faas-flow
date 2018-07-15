package main

import (
	"fmt"
	"github.com/s8sg/faaschain"
	"io/ioutil"
	"log"
	"os"
)

const (
	IMAGE_IP = "apollo13.jpg"
	IMAGE_OP = "apollo13-out.jpg"
)

func main() {
	chain := faaschain.NewFaaschain("http://127.0.0.1:8080")
	chain.Apply("colorization", map[string]string{"method": "post"}, nil).Apply("image-resizer", map[string]string{"method": "post"}, nil).Apply("facedetect", map[string]string{"method": "post"}, map[string][]string{"output": []string{"image"}})
	err := chain.Build()
	if err != nil {
		log.Fatalf("Failed to build chain: %v", err)
	}
	fmt.Printf("Chian Created %s\n", chain.GetDefinition())

	fmt.Printf("Loading image %s\n", IMAGE_IP)

	f, err := os.Open(IMAGE_IP)
	if err != nil {
		log.Fatalf("Failed to open file %s, error %v", IMAGE_IP, err)
	}

	fmt.Printf("Invoking chain \n")

	resp, err := chain.Invoke(nil, f)
	if err != nil {
		log.Fatalf("Failed to invoke chain, error %v", err)
	}

	data, err := ioutil.ReadAll(resp)
	if err != nil {
		log.Fatalf("Failed to read resp data, error %v", err)
	}

	err = ioutil.WriteFile(IMAGE_OP, data, 0644)
	if err != nil {
		log.Fatalf("Failed to write file %s, error %v", IMAGE_OP, err)
	}

	fmt.Printf("Written file :%s\n", IMAGE_OP)
}
