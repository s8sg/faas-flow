.PHONY: build

all: lib function


lib:
	./buildlib.sh

function:
	faas-cli build -f stack.yml	
