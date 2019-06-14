package sdk

import (
	"encoding/json"
)

type DagExporter struct {
	Id           string                   `json:"id"`
	StartNode    string                   `json:"start-node"`
	EndNode      string                   `json:"end-node"`
	HasBranch    bool                     `json:"has-branch"`
	HasEdge      bool                     `json:"has-edge"`
	ExecutionDag bool                     `json:"is-execution-dag"`
	Nodes        map[string]*NodeExporter `json:"nodes"`

	IsValid         bool   `json:"is-valid"`
	ValidationError string `json:"validation-error,omitempty"`
}

type NodeExporter struct {
	Id    string `json:"id"`
	Index int    `json:"node-index"`

	IsDynamic        bool `json:"is-dynamic"`
	IsCondition      bool `json:"is-condition"`
	IsForeach        bool `json:"is-foreach"`
	HasAgregator     bool `json:"has-aggregator"`
	HasSubAggregator bool `json:"has-sub-aggregator"`
	HasSubDag        bool `json:"has-subdag"`
	InDegree         int  `json:"in-degree"`
	OutDegree        int  `json:"out-degree"`

	SubDag          *DagExporter            `json:"subdag,omitempty"`
	ForeachDag      *DagExporter            `json:"foreachdag,omitempty"`
	ConditionalDags map[string]*DagExporter `json:"conditiondags,omitempty"`
	Operations      []*OperationExporter    `json:"operations,omitempty"`

	Childrens []string `json:"childrens,omitempty"`
}

type OperationExporter struct {
	IsMod              bool   `json:"is-mod"`
	IsFunction         bool   `json:"is-function"`
	IsCallback         bool   `json:"is-callback"`
	Name               string `json:"name"`
	HasResponseHandler bool   `json:"has-response-handler"`
	HasFailureHandler  bool   `json:"has-failure-handler"`
}

func exportOperation(exportOperation *OperationExporter, operation *Operation) {
	if operation.Mod != nil {
		exportOperation.IsMod = true
	}
	if operation.Function != "" {
		exportOperation.IsFunction = true
		exportOperation.Name = operation.Function
	}
	if operation.CallbackUrl != "" {
		exportOperation.IsCallback = true
		exportOperation.Name = operation.CallbackUrl[len(operation.CallbackUrl)-8:]
	}
	if operation.FailureHandler != nil {
		exportOperation.HasFailureHandler = true
	}
	if operation.OnResphandler != nil {
		exportOperation.HasResponseHandler = true
	}
}

func exportNode(exportNode *NodeExporter, node *Node) {
	exportNode.Id = node.Id
	exportNode.Index = node.index

	exportNode.IsDynamic = node.dynamic
	if node.GetCondition() != nil {
		exportNode.IsCondition = true
		for condition, sdag := range node.conditionalDags {
			if exportNode.ConditionalDags == nil {
				exportNode.ConditionalDags = make(map[string]*DagExporter)
			}
			exportNode.ConditionalDags[condition] = &DagExporter{}
			exportDag(exportNode.ConditionalDags[condition], sdag)
		}
	}
	if node.GetForEach() != nil {
		exportNode.IsForeach = true
		exportNode.ForeachDag = &DagExporter{}
		exportDag(exportNode.ForeachDag, node.subDag)
	}
	if node.aggregator != nil {
		exportNode.HasAgregator = true
	}
	if node.subAggregator != nil {
		exportNode.HasSubAggregator = true
	}
	if node.subDag != nil {
		exportNode.HasSubDag = true
		exportNode.SubDag = &DagExporter{}
		exportDag(exportNode.SubDag, node.subDag)
	}

	for _, operation := range node.operations {
		exportedOperation := &OperationExporter{}
		exportOperation(exportedOperation, operation)
		exportNode.Operations = append(exportNode.Operations, exportedOperation)
	}

	for _, snode := range node.children {
		exportNode.Childrens = append(exportNode.Childrens, snode.Id)
	}
}

func exportDag(exportDag *DagExporter, dag *Dag) {
	exportDag.Id = dag.Id
	if dag.initialNode != nil {
		exportDag.StartNode = dag.initialNode.Id
	}
	if dag.endNode != nil {
		exportDag.EndNode = dag.endNode.Id
	}
	exportDag.HasBranch = dag.hasBranch
	exportDag.HasEdge = dag.hasEdge
	exportDag.ExecutionDag = dag.executionFlow

	for nodeId, node := range dag.nodes {
		if exportDag.Nodes == nil {
			exportDag.Nodes = make(map[string]*NodeExporter)
		}
		exportedNode := &NodeExporter{}
		exportNode(exportedNode, node)
		exportDag.Nodes[nodeId] = exportedNode
	}
}

// GetDagDefinition generate pipeline DAG defintion as a json
func (pipeline *Pipeline) GetDagDefinition() string {
	root := &DagExporter{}

	// Validate the dag
	root.IsValid = true
	err := pipeline.Dag.Validate()
	if err != nil {
		root.IsValid = false
		root.ValidationError = err.Error()
	}

	exportDag(root, pipeline.Dag)
	encoded, _ := json.MarshalIndent(root, "", "    ")
	return string(encoded)
}
