.PHONY: build

all: build-lib build-template


build-lib:
	./ci/script/buildlib.sh

build-template:
	docker build -t faasflow:test template/faasflow 
