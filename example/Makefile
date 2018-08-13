all: build deploy-function

build:
	./build.sh

deploy-function:
	faas-cli deploy -f stack.yml

clean:
	faas-cli rm -f stack.yml
	rm -rf faas-colorization
	rm -rf cdn_faas
