.PHONY: build

all: build-lib build-template


build-lib:
	./ci/script/buildlib.sh

build-template:
	docker build -t faaschain:test template/faaschain 
