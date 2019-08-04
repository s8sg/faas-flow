package sdk

import ()

type Operation interface {
	GetId() string
	GetProperties() map[string][]string
}

type BlankOperation struct {
}

func (ops *BlankOperation) GetId() string {
	return "end"
}

func (ops *BlankOperation) GetProperties() map[string][]string {
	return make(map[string][]string)
}
