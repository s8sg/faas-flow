package sdk

import (
	"fmt"
	"log"
	"strings"
)

func generateDag(dag *Dag, sb *strings.Builder) {
	// generate nodes
	for _, node := range dag.nodes {

		log.Printf("%s - %s", dag.Id, node.Id)

		sb.WriteString(fmt.Sprintf("\n\tsubgraph cluster_%d {", node.index))
		nodeIndexStr := fmt.Sprintf("%d", node.index-1)
		if nodeIndexStr != node.Id {
			sb.WriteString(fmt.Sprintf("\n\t\tlabel=\"%d-%s\";", node.index, node.Id))
		} else {
			sb.WriteString(fmt.Sprintf("\n\t\tlabel=\"%d\";", node.index))
		}
		sb.WriteString("\n\t\tcolor=lightgrey;")
		sb.WriteString("\n\t\tstyle=rounded;\n")

		previousOperation := ""
		subdag := node.SubDag()
		if subdag != nil {
			generateDag(subdag, sb)
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
					sb.WriteString(fmt.Sprintf("\n\t\t\"%s\" [color=pink];",
						operationKey))
				case node.indegree == 0 && opsindex == 0:
					sb.WriteString(fmt.Sprintf("\n\t\t\"%s\" [color=lightblue];",
						operationKey))
				default:
					sb.WriteString(fmt.Sprintf("\n\t\t\"%s\" [color=lightgrey];",
						operationKey))
				}

				if previousOperation != "" {
					sb.WriteString(fmt.Sprintf("\n\t\t\"%s\" -> \"%s\" [label=\"1:1\" color=grey];",
						previousOperation, operationKey))
				}
				previousOperation = operationKey
			}

			sb.WriteString("\n\t}")

			relation := ""

			for _, child := range node.children {

				// TODO: Later change to check if 1:N

				relation = "1:1"
				operationStr := ""
				var operation sdk.Operation

				node = 
				for true {
					subDag = child.SubDag()
					if subdag != nil {
						operation = subdag.GetInitialNode()
					} else {
						operation = child.Operations()[0]
					}
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

				childOperationKey := fmt.Sprintf("%d.1-%s",
					child.index, operationStr)

				if node.GetForwarder(child.Id) == nil {
					relation = relation + " - nodata"
				}

				sb.WriteString(fmt.Sprintf("\n\t\"%s\" -> \"%s\" [label=\"%s\" color=grey];",
					previousOperation, childOperationKey, relation))
			}

			sb.WriteString("\n")
		}
	}
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

	generateDag(pipeline.Dag, &sb)

	sb.WriteString("}\n")
	return sb.String()

}
