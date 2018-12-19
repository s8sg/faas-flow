package sdk

import (
	"fmt"
	"strings"
)

// generateOperationKey generate a unique key for an operation
func generateOperationKey(dagId string, nodeIndex int, opsIndex int, operation *Operation, operationStr string) string {
	if operation != nil {
		switch {
		case operation.Function != "":
			operationStr = "func-" + operation.Function
		case operation.CallbackUrl != "":
			operationStr = "callback-" +
				operation.CallbackUrl[len(operation.CallbackUrl)-4:]
		default:
			operationStr = "modifier"
		}
	}
	operationKey := ""
	if dagId != "0" {
		operationKey = fmt.Sprintf("%s.%d.%d-%s", dagId, nodeIndex, opsIndex, operationStr)
	} else {
		operationKey = fmt.Sprintf("%d.%d-%s", nodeIndex, opsIndex, operationStr)
	}
	return operationKey
}

// generateConditionalDag generate dag element of a condition vertex
func generateConditionalDag(node *Node, dag *Dag, sb *strings.Builder, indent string) string {
	// Create a condition vertex
	conditionKey := generateOperationKey(dag.Id, node.index, 0, nil, "conditions")
	sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" [shape=Mdiamond];",
		indent, conditionKey))

	// Create a end operation vertex
	conditionEndKey := generateOperationKey(dag.Id, node.index, 0, nil, "end")
	sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" [shape=Msquare];",
		indent, conditionEndKey))

	// Create condition graph
	for condition, conditionDag := range node.GetAllConditionalDags() {
		nextOperationNode := conditionDag.GetInitialNode()
		nextOperationDag := conditionDag

		// Find out the first operation on a subdag
		for nextOperationNode.SubDag() != nil && !nextOperationNode.Dynamic() {
			nextOperationDag = nextOperationNode.SubDag()
			nextOperationNode = nextOperationDag.GetInitialNode()
		}
		operationKey := ""
		if nextOperationNode.Dynamic() {
			if nextOperationNode.GetCondition() != nil {
				operationKey = generateOperationKey(nextOperationDag.Id, nextOperationNode.index, 0, nil, "conditions")
			}
			if nextOperationNode.GetForEach() != nil {
				operationKey = generateOperationKey(nextOperationDag.Id, nextOperationNode.index, 0, nil, "foreach")
			}
		} else {
			operation := nextOperationNode.Operations()[0]
			operationKey = generateOperationKey(nextOperationDag.Id, nextOperationNode.index, 1, operation, "")
		}
		sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" -> \"%s\" [label=%s color=grey];",
			indent, conditionKey, operationKey, condition))

		previousOperation := generateDag(conditionDag, sb, indent+"\t")

		sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" -> \"%s\" [label=\"1:1\" color=grey];",
			indent, previousOperation, conditionEndKey))
	}

	return conditionEndKey
}

// generateForeachDag generate dag element of a foreach vertex
func generateForeachDag(node *Node, dag *Dag, sb *strings.Builder, indent string) string {
	subdag := node.SubDag()

	// Create a foreach operation vertex
	foreachKey := generateOperationKey(dag.Id, node.index, 0, nil, "foreach")
	sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" [shape=Mdiamond];",
		indent, foreachKey))

	// Create a end operation vertex
	foreachEndKey := generateOperationKey(dag.Id, node.index, 0, nil, "end")
	sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" [shape=Msquare];",
		indent, foreachEndKey))

	// Create Foreach Graph
	{
		nextOperationNode := subdag.GetInitialNode()
		nextOperationDag := subdag

		// Find out the first operation on a subdag
		for nextOperationNode.SubDag() != nil && !nextOperationNode.Dynamic() {
			nextOperationDag = nextOperationNode.SubDag()
			nextOperationNode = nextOperationDag.GetInitialNode()
		}

		operationKey := ""
		if nextOperationNode.Dynamic() {
			if nextOperationNode.GetCondition() != nil {
				operationKey = generateOperationKey(nextOperationDag.Id, nextOperationNode.index, 0, nil, "conditions")
			}
			if nextOperationNode.GetForEach() != nil {
				operationKey = generateOperationKey(nextOperationDag.Id, nextOperationNode.index, 0, nil, "foreach")
			}

		} else {
			operation := nextOperationNode.Operations()[0]
			operationKey = generateOperationKey(nextOperationDag.Id, nextOperationNode.index, 1, operation, "")
		}

		sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" -> \"%s\" [label=\"1:N\" color=grey];",
			indent, foreachKey, operationKey))

		previousOperation := generateDag(subdag, sb, indent+"\t")

		sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" -> \"%s\" [label=\"N:1\" color=grey];",
			indent, previousOperation, foreachEndKey))
	}

	return foreachEndKey
}

// generateDag populate a string buffer for a dag and returns the last operation ID
func generateDag(dag *Dag, sb *strings.Builder, indent string) string {
	lastOperation := ""
	// generate nodes
	for _, node := range dag.nodes {

		if dag.Id != "0" {
			sb.WriteString(fmt.Sprintf("\n%ssubgraph cluster_%d {", indent, node.index))
		} else {
			sb.WriteString(fmt.Sprintf("\n%ssubgraph cluster_%s_%d {", indent, dag.Id, node.index))
		}

		nodeIndexStr := fmt.Sprintf("%d", node.index-1)

		if nodeIndexStr != node.Id {
			if dag.Id != "0" {
				sb.WriteString(fmt.Sprintf("\n%s\tlabel=\"%s.%d-%s\";", indent, dag.Id, node.index, node.Id))
			} else {
				sb.WriteString(fmt.Sprintf("\n%s\tlabel=\"%d-%s\";", indent, node.index, node.Id))

			}
		} else {
			if dag.Id != "0" {
				sb.WriteString(fmt.Sprintf("\n%s\tlabel=\"%s-%d\";", indent, dag.Id, node.index))
			} else {

				sb.WriteString(fmt.Sprintf("\n%s\tlabel=\"%d\";", indent, node.index))

			}
		}
		sb.WriteString(fmt.Sprintf("\n%s\tcolor=lightgrey;", indent))
		sb.WriteString(fmt.Sprintf("\n%s\tstyle=rounded;\n", indent))

		previousOperation := ""

		subdag := node.SubDag()
		if node.Dynamic() {
			if node.GetCondition() != nil {
				previousOperation = generateConditionalDag(node, dag, sb, indent)
			}
			if node.GetForEach() != nil {
				previousOperation = generateForeachDag(node, dag, sb, indent)
			}
		} else if subdag != nil {
			previousOperation = generateDag(subdag, sb, indent+"\t")
		} else {
			for opsindex, operation := range node.Operations() {
				operationKey := generateOperationKey(dag.Id, node.index, opsindex+1, operation, "")

				switch {
				case len(node.children) == 0 &&
					opsindex == len(node.Operations())-1:
					sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" [color=pink];",
						indent, operationKey))
				case node.indegree == 0 && opsindex == 0:
					sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" [color=lightblue];",
						indent, operationKey))
				default:
					sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" [color=lightgrey];",
						indent, operationKey))
				}

				if previousOperation != "" {
					sb.WriteString(fmt.Sprintf("\n%s\t\"%s\" -> \"%s\" [label=\"1:1\" color=grey];",
						indent, previousOperation, operationKey))
				}
				previousOperation = operationKey
			}
		}

		sb.WriteString(fmt.Sprintf("\n%s}\n", indent))

		if node.children != nil {
			for _, child := range node.children {

				var operation *Operation

				nextOperationNode := child
				nextOperationDag := dag

				// Find out the first operation on a subdag
				for nextOperationNode.SubDag() != nil && !nextOperationNode.Dynamic() {
					nextOperationDag = nextOperationNode.SubDag()
					nextOperationNode = nextOperationDag.GetInitialNode()
				}

				childOperationKey := ""
				if nextOperationNode.Dynamic() {
					if nextOperationNode.GetCondition() != nil {
						childOperationKey = generateOperationKey(nextOperationDag.Id, nextOperationNode.index, 0, nil, "conditions")
					}
					if nextOperationNode.GetForEach() != nil {
						childOperationKey = generateOperationKey(nextOperationDag.Id, nextOperationNode.index, 0, nil, "foreach")
					}

				} else {
					operation = nextOperationNode.Operations()[0]
					childOperationKey = generateOperationKey(nextOperationDag.Id, nextOperationNode.index, 1, operation, "")
				}

				if previousOperation != "" {
					if node.GetForwarder(child.Id) == nil {
						sb.WriteString(
							fmt.Sprintf("\n%s\"%s\" -> \"%s\" [style=dashed label=\"1:1\" color=grey];",
								indent, previousOperation, childOperationKey))
					} else {
						sb.WriteString(fmt.Sprintf("\n%s\"%s\" -> \"%s\" [label=\"1:1\" color=grey];",
							indent, previousOperation, childOperationKey))
					}
				}
			}
		} else {
			lastOperation = previousOperation
		}

		sb.WriteString("\n")
	}
	return lastOperation
}

// MakeDotGraph create a dot graph of the pipeline
func (pipeline *Pipeline) MakeDotGraph() string {

	var sb strings.Builder

	sb.WriteString("digraph depgraph {")
	sb.WriteString("\n\trankdir=TB;")
	sb.WriteString("\n\tsplines=curved;")
	sb.WriteString("\n\tfontname=\"Courier New\";")
	sb.WriteString("\n\tfontcolor=grey;")

	sb.WriteString("\n\tnode [style=filled fontname=\"Courier\" fontcolor=black]\n")

	generateDag(pipeline.Dag, &sb, "\t")

	sb.WriteString("}\n")
	return sb.String()

}
