FROM golang:1.10.4-alpine3.8 as build

WORKDIR /go/src/github.com/s8sg/faas-flow
COPY . .

RUN (cd sdk && go build)
RUN (cd sdk/executor && go build)
RUN (cd sdk/exporter && go build)
RUN go build
