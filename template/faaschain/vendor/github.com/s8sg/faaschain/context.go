package faaschain

// Context execution context and execution state
type Context struct {
	requestId    string
	phase        int
	stateManager StateManager
	State        string
}

// StateManager for State Manager
type StateManager interface {
	Set(key string, value interface{}) error
	Get(key string) (interface{}, error)
	Del(key string) error
}

const (
	StateSuccess = "success"
	StateFailure = "failure"
	StateOngoing = "ongoing"
)

// CreateContext create request context (used by template)
func CreateContext(fchain *Fchain) *Context {
	context := &Context{}
	context.requestId = fchain.id
	context.phase = fchain.chain.ExecutionPosition + 1
	context.State = StateOngoing
	return context
}

// SetStateManager sets the state manager
func (context *Context) SetStateManager(state StateManager) {
	context.stateManager = state
}

// GetRequestId returns the request id
func (context *Context) GetRequestId() string {
	return context.requestId
}

// GetPhase return the phase no
func (context *Context) GetPhase() int {
	return context.phase
}

// Set put a value in the context with StateManager
func (context *Context) Set(key string, data interface{}) error {
	return context.stateManager.Set(key, data)
}

// Get retrive a value from the context with StateManager
func (context *Context) Get(key string) (interface{}, error) {
	data, err := context.stateManager.Get(key)
	return data, err
}

// Del deletes a value from the context with StateManager
func (context *Context) Del(key string) error {
	return context.stateManager.Del(key)
}
