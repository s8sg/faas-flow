#!/bin/bash

echo "Get faas-colorization"
[ ! "$(ls | grep faas-colorization)" ] && git clone https://github.com/alexellis/faas-colorization.git
echo "Building and Deploying faas-colorization"
faas-cli build -f stack.yml --regex colorization
faas-cli deploy -f stack.yml --regex colorization

echo "Get facedetect"
[ ! "$(ls | grep open-faas-functions)" ] && git clone https://github.com/nicholasjackson/open-faas-functions.git
echo "Building and Deploying facedetect"
faas-cli template pull https://github.com/s8sg/open-faas-templates
faas-cli build -f stack.yml --regex facedetect
faas-cli deploy -f stack.yml --regex facedetect

echo "Get image-resizer"
[ ! "$(ls | grep cdn_faas)" ] && git clone https://github.com/s8sg/cdn_faas.git
echo "Building and Deploying image-resizer"
faas-cli build -f stack.yml --regex image-resizer
faas-cli deploy -f stack.yml --regex image-resizer

echo "Done"
