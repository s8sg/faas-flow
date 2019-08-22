package sdk

type Operation interface {
	GetId() string
	GetProperties() map[string][]string
	Execute([]byte) ([]byte, error)
	Encode() ([]byte, error)
	Decode([]byte) error
}

type BlankOperation struct {
}

func (ops *BlankOperation) GetId() string {
	return "end"
}

func (ops *BlankOperation) GetProperties() map[string][]string {
	return make(map[string][]string)
}

func (ops *BlankOperation) Encode() ([]byte, error) {
	return nil, nil
}

func (ops *BlankOperation) Decode([]byte) error {
	return nil
}

func (ops *BlankOperation) Execute([]byte) ([]byte, error) {
	return nil, nil
}
