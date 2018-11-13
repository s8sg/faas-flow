#!/bin/bash

faas-cli rm -f stack.yml

docker service rm minio
