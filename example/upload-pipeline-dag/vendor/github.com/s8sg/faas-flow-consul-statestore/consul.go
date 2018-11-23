package ConsulStateStore

import (
	"fmt"
	consul "github.com/hashicorp/consul/api"
	faasflow "github.com/s8sg/faas-flow"
	"strconv"
)

type ConsulStateStore struct {
	consulKeyPath string
	consulClient  *consul.Client
	kv            *consul.KV

	RetryCount int
}

func GetConsulStateStore(address, datacenter string) (faasflow.StateStore, error) {

	consulST := &ConsulStateStore{}

	config := consul.DefaultConfig()
	if address != "" {
		config.Address = address
	}
	if datacenter != "" {
		config.Datacenter = datacenter
	}

	cc, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}
	consulST.consulClient = cc
	consulST.kv = cc.KV()

	consulST.RetryCount = 10

	return consulST, nil
}

// Configure
func (consulStore *ConsulStateStore) Configure(flowName string, requestId string) {
	consulStore.consulKeyPath = fmt.Sprintf("faasflow/%s/%s", flowName, requestId)
}

// Init (Called only once in a request)
func (consulStore *ConsulStateStore) Init() error {
	return nil
}

// Create
func (consulStore *ConsulStateStore) Create(vertexs []string) error {

	for _, vertex := range vertexs {
		key := fmt.Sprintf("%s/%s", consulStore.consulKeyPath, vertex)
		p := &consul.KVPair{Key: key, Value: []byte("0")}
		_, err := consulStore.kv.Put(p, nil)
		if err != nil {
			return fmt.Errorf("failed to create vertex %s, error %v", vertex, err)
		}
	}
	return nil
}

// IncrementCounter
func (consulStore *ConsulStateStore) IncrementCounter(vertex string) (int, error) {
	count := 0
	key := fmt.Sprintf("%s/%s", consulStore.consulKeyPath, vertex)
	for i := 0; i < consulStore.RetryCount; i++ {
		pair, _, err := consulStore.kv.Get(key, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to get vertex %s, error %v", vertex, err)
		}
		modifyIndex := pair.ModifyIndex
		counter, err := strconv.Atoi(string(pair.Value))
		if err != nil {
			return 0, fmt.Errorf("failed to convert counter for %s, error %v", vertex, err)
		}

		count := counter + 1
		counterStr := fmt.Sprintf("%d", count)

		p := &consul.KVPair{Key: key, Value: []byte(counterStr), ModifyIndex: modifyIndex}
		_, err = consulStore.kv.Put(p, nil)
		if err != nil {
			continue
		}
	}
	return count, nil
}

// SetState set state of pipeline
func (consulStore *ConsulStateStore) SetState(state bool) error {
	key := fmt.Sprintf("%s/state", consulStore.consulKeyPath)
	stateStr := "false"
	if state {
		stateStr = "true"
	}

	p := &consul.KVPair{Key: key, Value: []byte(stateStr)}
	_, err := consulStore.kv.Put(p, nil)
	if err != nil {
		return fmt.Errorf("failed to set state, error %v", err)
	}
	return nil
}

// GetState set state of the pipeline
func (consulStore *ConsulStateStore) GetState() (bool, error) {
	key := fmt.Sprintf("%s/state", consulStore.consulKeyPath)
	pair, _, err := consulStore.kv.Get(key, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get state, error %v", err)
	}
	state := false
	if string(pair.Value) == "true" {
		state = true
	}
	return state, nil
}

// Cleanup (Called only once in a request)
func (consulStore *ConsulStateStore) Cleanup() error {
	_, err := consulStore.kv.DeleteTree(consulStore.consulKeyPath, nil)
	if err != nil {
		return fmt.Errorf("error removing %s, error %v", consulStore.consulKeyPath, err)
	}
	return nil
}
