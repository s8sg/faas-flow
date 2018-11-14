package main

import (
	"fmt"
)

// json to encode
type requestEmbedDataStore struct {
	store map[string]string
}

// CreateDataStore creates a new requestEmbedDataStore
func createDataStore() *requestEmbedDataStore {
	rstore := &requestEmbedDataStore{}
	rstore.store = make(map[string]string)
	return rstore
}

// RetriveDataStore creates a store manager from a map
func retriveDataStore(store map[string]string) *requestEmbedDataStore {
	rstore := &requestEmbedDataStore{}
	rstore.store = store
	return rstore
}

// Init initialize the storemanager with requestId and flowname
func (rstore *requestEmbedDataStore) Init(flowName string, requestId string) error {
	return nil
}

// Set sets a value (implement DataStore)
func (rstore *requestEmbedDataStore) Set(key string, value string) error {
	rstore.store[key] = value
	return nil
}

// Get gets a value (implement DataStore)
func (rstore *requestEmbedDataStore) Get(key string) (string, error) {
	value, ok := rstore.store[key]
	if !ok {
		return "", fmt.Errorf("No field name %s", key)
	}
	return value, nil
}

// Del delets a value (implement DataStore)
func (rstore *requestEmbedDataStore) Del(key string) error {
	if _, ok := rstore.store[key]; ok {
		delete(rstore.store, key)
	}
	return nil
}
