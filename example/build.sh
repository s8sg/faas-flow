#!/bin/bash

echo "Get image-resizer"
[ ! "$(ls | grep cdn_faas)" ] && git clone https://github.com/s8sg/cdn_faas.git

echo "Get faas-colorization"
[ ! "$(ls | grep faas-colorization)" ] && git clone https://github.com/alexellis/faas-colorization.git

echo "Get face-detect"
[ ! "$(ls | grep open-faas-functions)" ] && git clone https://github.com/nicholasjackson/open-faas-functions.git

echo "Get opencv template"
faas-cli template pull https://github.com/s8sg/open-faas-templates

echo "Get faaschain template"
faas-cli template pull https://github.com/s8sg/faaschain

faas-cli build -f stack.yml
