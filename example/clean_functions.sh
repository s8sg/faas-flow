#!/bin/sh

rm -rf faas-colorization
rm -rf open-faas-functions
rm -rf cdn_faas

faas-cli rm colorization
faas-cli rm facedetect
faas-cli rm image-resizer
