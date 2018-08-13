#!/bin/bash

echo "Get image-resizer"
[ ! "$(ls | grep cdn_faas)" ] && git clone https://github.com/s8sg/cdn_faas.git

echo "Get faas-colorization"
[ ! "$(ls | grep faas-colorization)" ] && git clone https://github.com/alexellis/faas-colorization.git

echo "Get faaschain template"
faas-cli template pull https://github.com/s8sg/faaschain

faas-cli build -f stack.yml
