package sdk

import (
	"fmt"
	"strings"
)

func generateDag(dag *Dag, sb *strings.Builder, indent string) string {
	lastOperation := ""
	// generate nodes
	for _, node := range dag.nodes {

		sb.WriteString(fmt.Sprintf("\n%ssubgraph cluster_%d {", indent, node.index))
		nodeIndexStr := fmt.Sprintf("%d", node.index-1)
		if nodeIndexStr != node.Id {
			sb.WriteString(fmt.Sprintf("\n%s\tlabel=\"%d-%s\";", indent, node.index, node.Id))
		} else {
			sb.WriteString(fmt.Sprintf("\n%s\tlabel=\"%d\";", indent, node.index))
		}
		sb.WriteString(fmt.Sprintf("\n%s\tcolor=lightgrey;", indent))
		sb.WriteString(fmt.Sprintf("\n%s\tstyle=rounded;\n", indent))

		previousOperation := ""

		subdag := node.SubDag()
		if subdag != nil {
			previousOperation = generateDag(subdag, sb, indent+"\t")
		} else {
			for opsindex, operation := range node.Operations() {
				operationStr := ""
				switch {
				case operation.Function != "":
					operationStr = "func-" + operation.Function
				case operation.CallbackUrl != "":
					operationStr = "callback-" +
						operation.CallbackUrl[len(operation.CallbackUrl)-4:]
				default:
					operationStr = "modifier"
				}
				operationKey := fmt.Sprintf("%s.%d.%d-%s", dag.Id, node.index, opsindex+1, operationStr)

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

		relation := ""

		if node.children != nil {
			for _, child := range node.children {

				// TODO: Later change to check if 1:N

				relation = "1:1"
				operationStr := ""
				var operation *Operation

				nextOperationNode := child
				nextOperationDag := dag

				if nextOperationNode.SubDag() != nil {
					for nextOperationNode.SubDag() != nil {
						nextOperationDag = nextOperationNode.SubDag()
						nextOperationNode = nextOperationDag.GetInitialNode()
					}
					operation = nextOperationNode.Operations()[0]
				} else {
					operation = nextOperationNode.Operations()[0]
				}

				switch {
				case operation.Function != "":
					operationStr = "func-" + operation.Function
				case operation.CallbackUrl != "":
					operationStr = "callback-" +
						operation.CallbackUrl[len(operation.CallbackUrl)-4:]
				default:
					operationStr = "modifier"
				}

				childOperationKey := fmt.Sprintf("%s.%d.1-%s",
					nextOperationDag.Id, nextOperationNode.index, operationStr)

				if node.GetForwarder(child.Id) == nil {
					relation = relation + " - nodata"
				}

				if previousOperation != "" {
					sb.WriteString(fmt.Sprintf("\n%s\"%s\" -> \"%s\" [label=\"%s\" color=grey];",
						indent, previousOperation, childOperationKey, relation))
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
