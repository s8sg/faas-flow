.PHONY: build

all: build-lib build-func deploy


build-lib:
	./buildlib.sh

build-func:
	faas-cli build -f stack.yml

deploy: 
	faas-cli deploy -f stack.yml
