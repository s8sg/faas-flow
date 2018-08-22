all: build deploy-function

build:
	./build.sh

deploy-function:
	faas-cli deploy -f stack.yml

clean:
