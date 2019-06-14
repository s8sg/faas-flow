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
		if opsIndex != 0 {
			operationKey = fmt.Sprintf("%s.%d.%d-%s", dagId, nodeIndex, opsIndex, operationStr)
		} else {
			operationKey = fmt.Sprintf("%s.%d-%s", dagId, nodeIndex, operationStr)
		}
	} else {
		operationKey = fmt.Sprintf("%d.%d-%s", nodeIndex, opsIndex, operationStr)
	}
	return operationKey
}

// generateConditionalDag generate dag element of a condition vertex
func generateConditionalDag(node *Node, dag *Dag, sb *strings.Builder, indent string) string {
	// Create a condition vertex
	conditionKey := generateOperationKey(dag.Id, node.index, 0, nil, "conditions")
	sb.WriteString(fmt.Sprintf("\n%s\"%s\" [shape=diamond style=filled color=\"#f9af4d\"];",
		indent, conditionKey))

	// Create a end operation vertex
	conditionEndKey := generateOperationKey(dag.Id, node.index, 0, nil, "end")
	sb.WriteString(fmt.Sprintf("\n%s\"%s\" [shape=invhouse style=filled color=pink];",
		indent, conditionEndKey))

	// Create condition graph
	for condition, conditionDag := range node.GetAllConditionalDags() {
		nextOperationNode := conditionDag.GetInitialNode()
		nextOperationDag := conditionDag

		// Find out the first operation on a subdag (recursively)
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
		sb.WriteString(fmt.Sprintf("\n%s\"%s\" -> \"%s\" [label=%s color=black];",
			indent, conditionKey, operationKey, condition))

		sb.WriteString(fmt.Sprintf("\n%ssubgraph cluster_%s {", indent, condition))

		sb.WriteString(fmt.Sprintf("\n%slabel=\"%s.%d-%s\";", indent+"\t", dag.Id, node.index, condition))
		sb.WriteString(fmt.Sprintf("\n%scolor=grey;", indent+"\t"))
		sb.WriteString(fmt.Sprintf("\n%sstyle=rounded;\n", indent+"\t"))
		sb.WriteString(fmt.Sprintf("\n%sstyle=dashed;\n", indent+"\t"))

		previousOperation := generateDag(conditionDag, sb, indent+"\t")

		sb.WriteString(fmt.Sprintf("\n%s}", indent))

		sb.WriteString(fmt.Sprintf("\n%s\"%s\" -> \"%s\" [color=\"#152730\"];",
			indent, previousOperation, conditionEndKey))
	}

	return conditionEndKey
}

// generateForeachDag generate dag element of a foreach vertex
func generateForeachDag(node *Node, dag *Dag, sb *strings.Builder, indent string) string {
	subdag := node.SubDag()

	// Create a foreach operation vertex
	foreachKey := generateOperationKey(dag.Id, node.index, 0, nil, "foreach")
	sb.WriteString(fmt.Sprintf("\n%s\"%s\" [shape=diamond style=filled color=\"#f9af4d\"];",
		indent, foreachKey))

	// Create a end operation vertex
	foreachEndKey := generateOperationKey(dag.Id, node.index, 0, nil, "end")
	sb.WriteString(fmt.Sprintf("\n%s\"%s\" [shape=doublecircle style=filled color=pink];",
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

		sb.WriteString(fmt.Sprintf("\n%s\"%s\" -> \"%s\" [label=\"1:N\" color=\"#152730\"];",
			indent, foreachKey, operationKey))

		previousOperation := generateDag(subdag, sb, indent+"\t")

		sb.WriteString(fmt.Sprintf("\n%s\"%s\" -> \"%s\" [label=\"N:1\" color=\"#152730\"];",
			indent, previousOperation, foreachEndKey))
	}

	return foreachEndKey
}

// generateDag populate a string buffer for a dag and returns the last operation ID
func generateDag(dag *Dag, sb *strings.Builder, indent string) string {
	lastOperation := ""
	// generate nodes
	for _, node := range dag.nodes {

		previousOperation := ""

		if node.Dynamic() {
			if node.GetCondition() != nil {
				previousOperation = generateConditionalDag(node, dag, sb, indent)
			}
			if node.GetForEach() != nil {
				previousOperation = generateForeachDag(node, dag, sb, indent)
			}
		} else {

			sb.WriteString(fmt.Sprintf("\n%ssubgraph cluster_%d {", indent, node.index))

			nodeIndexStr := fmt.Sprintf("%d", node.index-1)

			if nodeIndexStr != node.Id {
				if dag.Id != "0" {
					sb.WriteString(fmt.Sprintf("\n%slabel=\"%s.%d-%s\";", indent+"\t", dag.Id, node.index, node.Id))
				} else {
					sb.WriteString(fmt.Sprintf("\n%slabel=\"%d-%s\";", indent+"\t", node.index, node.Id))

				}
			} else {
				if dag.Id != "0" {
					sb.WriteString(fmt.Sprintf("\n%slabel=\"%s-%d\";", indent+"\t", dag.Id, node.index))
				} else {

					sb.WriteString(fmt.Sprintf("\n%slabel=\"%d\";", indent+"\t", node.index))

				}
			}
			sb.WriteString(fmt.Sprintf("\n%scolor=grey;", indent+"\t"))
			sb.WriteString(fmt.Sprintf("\n%sstyle=rounded;\n", indent+"\t"))
		}

		subdag := node.SubDag()
		if subdag != nil {
			previousOperation = generateDag(subdag, sb, indent+"\t")
		} else {
			for opsindex, operation := range node.Operations() {
				operationKey := generateOperationKey(dag.Id, node.index, opsindex+1, operation, "")

				sb.WriteString(fmt.Sprintf("\n%s\"%s\" [shape=rectangle color=\"#68b8e2\" style=filled];",
					indent+"\t", operationKey))

				if previousOperation != "" {
					sb.WriteString(fmt.Sprintf("\n%s\"%s\" -> \"%s\" [color=\"#152730\"];",
						indent+"\t", previousOperation, operationKey))
				}
				previousOperation = operationKey
			}
		}

		if !node.Dynamic() {
			sb.WriteString(fmt.Sprintf("\n%s}\n", indent))
		}

		if node.children != nil {
			for _, child := range node.children {

				var operation *Operation

				nextOperationNode := child
				nextOperationDag := dag

				// Find out the first operation on a subdag (recursively)
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
					sb.WriteString(fmt.Sprintf("\n%s\"%s\" -> \"%s\" [color=\"#152730\"];",
						indent, previousOperation, childOperationKey))
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

	indent := "\t"
	sb.WriteString("digraph depgraph {")
	sb.WriteString(fmt.Sprintf("\n%srankdir=TD;", indent))
	sb.WriteString(fmt.Sprintf("\n%spack=1;", indent))
	sb.WriteString(fmt.Sprintf("\n%spad=0;", indent))
	sb.WriteString(fmt.Sprintf("\n%snodesep=0;", indent))
	sb.WriteString(fmt.Sprintf("\n%sranksep=0;", indent))
	sb.WriteString(fmt.Sprintf("\n%ssplines=curved;", indent))
	sb.WriteString(fmt.Sprintf("\n%sfontname=\"Courier New\";", indent))
	sb.WriteString(fmt.Sprintf("\n%sfontcolor=\"#44413b\";", indent))

	sb.WriteString(fmt.Sprintf("\n%snode [style=filled fontname=\"Courier\" fontcolor=black]\n", indent))

	generateDag(pipeline.Dag, &sb, indent)

	sb.WriteString("}\n")
	return sb.String()
}
