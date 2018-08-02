.PHONY: build

all: build-lib build-template


build-lib:
	./buildlib.sh

build-template:
	docker build -t faaschain:test template/faaschain 
