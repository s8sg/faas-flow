package main

import (
	"fmt"
	"handler/function/sdk"
)

func GetChain(string name) (*sdk.Request, error) {
	return nil, fmt.Errorf("store is not implemented")
}

func SaveChain(request *sdk.Request) error {
	return fmt.Errorf("store is not implemented")
}

func RemoveChain(string name) error {
	return fmt.Errorf("store is not implemented")
}
