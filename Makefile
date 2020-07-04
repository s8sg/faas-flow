.PHONY: build
all: build-template

build-template:
	docker build -t faas-flow:test template/faas-flow
