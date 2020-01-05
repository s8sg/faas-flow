package executor

import (
	"fmt"
)

// json to encode
type requestEmbedDataStore struct {
	store map[string][]byte
}

// CreateDataStore creates a new requestEmbedDataStore
func createDataStore() *requestEmbedDataStore {
	rstore := &requestEmbedDataStore{}
	rstore.store = make(map[string][]byte)
	return rstore
}

// retrieveDataStore creates a store manager from a map
func retrieveDataStore(store map[string][]byte) *requestEmbedDataStore {
	rstore := &requestEmbedDataStore{}
	rstore.store = store
	return rstore
}

// Configure Configure with requestId and flowname
func (rstore *requestEmbedDataStore) Configure(flowName string, requestId string) {

}

// Init initialize the storemanager (called only once in a request span)
func (rstore *requestEmbedDataStore) Init() error {
	return nil
}

// Set sets a value (implement DataStore)
func (rstore *requestEmbedDataStore) Set(key []byte, value []byte) error {
	rstore.store[string(key)] = value
	return nil
}

// Get gets a value (implement DataStore)
func (rstore *requestEmbedDataStore) Get(key []byte) ([]byte, error) {
	value, ok := rstore.store[string(key)]
	if !ok {
		return nil, fmt.Errorf("no field name %s", key)
	}
	return value, nil
}

// Del delets a value (implement DataStore)
func (rstore *requestEmbedDataStore) Del(key []byte) error {
	if _, ok := rstore.store[string(key)]; ok {
		delete(rstore.store, string(key))
	}
	return nil
}

// Cleanup
func (rstore *requestEmbedDataStore) Cleanup() error {
	return nil
}
