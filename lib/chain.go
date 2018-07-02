package lib

type Chain struct {
	Phases            []*Phase // Phases that will be executed in async
	ExecutionPosition int      // Position of Executor
}
