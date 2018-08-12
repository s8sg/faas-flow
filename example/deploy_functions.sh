#!/bin/bash
echo "Deploying stack"
faas-cli deploy -f stack.yml

echo "Done"
