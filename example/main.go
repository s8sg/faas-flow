package main

import "fmt"
import "log"
import "github.com/s8sg/faaschain"

func main() {
	chain := faaschain.NewFaaschain("127.0.0.1:8080")
	err := chain.Build()
	if err != nil {
		log.Fatalf("Failed to build chain: %v", err)
	}
	fmt.Printf("Chian Created %s", chain.GetDefinition())
}
