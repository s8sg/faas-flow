#!/bin/bash

echo "Get image-resizer"
[ ! "$(ls | grep cdn_faas)" ] && git clone https://github.com/s8sg/cdn_faas.git
echo "Building image-resizer"
faas-cli build -f stack.yml --regex image-resizer

echo "Get faas-colorization"
[ ! "$(ls | grep faas-colorization)" ] && git clone https://github.com/alexellis/faas-colorization.git
echo "Building faas-colorization"
faas-cli build -f stack.yml --regex colorization

echo "Get faaschain template"
faas-cli template pull https://github.com/s8sg/faaschain
echo "Building upload-chain"
faas-cli build -f stack.yml --regex upload-chain 
echo "Building upload-chain"
faas-cli build -f stack.yml --regex upload-chain-async

echo "Done"
