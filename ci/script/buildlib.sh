#!/bin/bash

# Run a gofmt and exclude all vendored code.
test -z "$(gofmt -l $(find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./example/*" -not -path "./faaschain/*" -not -path "./" ))" || { echo "Run \"gofmt -s -w\" on your Golang code"; exit 1; }

go test ./chain.go ./chain_test.go -cover
