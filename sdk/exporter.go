package sdk

type DagExporter struct {
	Id           string                   `json: "id"`
	Nodes        map[string]*NodeExporter `json: "nodes"`
	StartNode    string                   `json: "start-node"`
	EndNode      string                   `json: "end-node"`
	HasBranch    bool                     `json: "has-branch"`
	HasEdge      bool                     `json: "has-edge"`
	ExecutionDag bool                     `json: "is-execution-dag"`
}

type NodeExporter struct {
	Id    string `json: "id"`
	Index int    `json: "node-index"`

	IsDynamic       bool   `json: "is-dynamic"`
	IsCondition     bool   `json: "is-condition"`
	IsForeach       bool   `json: "is-foreach"`
	HasAgregator    bool   `json: "has-aggregator"`
	HasSubAgregator bool   `json: "has-sub-aggregator"`
	HasSubgraph     string `json: "has-subdag"`
	InDegree        int    `json: "in-degree"`
	OutDegree       int    `json: "out-degree"`

	SubDag          *DagExporter            `json: "subdag, omitempty"`
	ForeachDag      *DagExporter            `json: "foreachdag, omitempty"`
	ConditionalDags map[string]*DagExporter `json: "conditiondags, omitempty"`
	Operations      []*OperationExporter    `json: "operations, omitempty"`

	Childrens []string `json: "clindrens"`
}

type OperationExporter struct {
	IsMod              bool   `json: "is-mod"`
	IsFunction         bool   `json: "is-function"`
	Name               string `json: "name"`
	HasResponseHandler bool   `json: "has-response-handler"`
	HasFailureHandler  bool   `json: "has-failure-handler"`
}

func (pipeline *Pipeline) GetDagDefinition() string {
	return ""
}
