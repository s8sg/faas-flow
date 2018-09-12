#!/bin/bash

# Run a gofmt and exclude all vendored code.
test -z "$(gofmt -l $(find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./example/*" -not -path "./template/*" -not -path "./doc/*" -not -path "./ci/" -not -path "./" ))" || { echo "Run \"gofmt -s -w\" on your Golang code"; exit 1; }

go test ./workflow.go ./context.go ./workflow_test.go -cover
