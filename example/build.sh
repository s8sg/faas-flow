#!/bin/bash

echo "Get image-resizer"
[ ! "$(ls | grep cdn_faas)" ] && git clone https://github.com/s8sg/cdn_faas.git

echo "Get faas-colorization"
[ ! "$(ls | grep faas-colorization)" ] && git clone https://github.com/alexellis/faas-colorization.git

echo "Get face-detect"
[ ! "$(ls | grep facedetect-openfaas)" ] && git clone https://github.com/alexellis/facedetect-openfaas.git

echo "Get opencv template"
faas-cli template pull https://github.com/alexellis/opencv-openfaas-template

echo "Get faasflow template"
faas-cli template pull https://github.com/s8sg/faasflow

echo "Get faas default template"
faas-cli template pull

faas-cli build -f stack.yml

