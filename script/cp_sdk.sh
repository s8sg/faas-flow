#!/bin/bash
(cd .. && rm -rf template/faas-flow/vendor/github.com/s8sg/faas-flow/*)
(cd .. && cp -r sdk workflow.go faas_operation.go template/faas-flow/vendor/github.com/s8sg/faas-flow/)
