FROM openfaas/of-watchdog:0.7.7 as watchdog
FROM golang:1.14.3-alpine3.11 as build

COPY --from=watchdog /fwatchdog /usr/bin/fwatchdog
RUN chmod +x /usr/bin/fwatchdog

RUN mkdir -p /go/src/handler
WORKDIR /go/src/handler

RUN apk update && \
    apk --no-cache add \
        gcc \
        musl-dev

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY . .

# Run a gofmt and exclude all vendored code.
RUN test -z "$(gofmt -l $(find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./function/vendor/*"))" || { echo "Run \"gofmt -s -w\" on your Golang code"; exit 1; }

RUN CGO_ENABLED=0 GOOS=linux \
    go build --ldflags "-s -w" -a -installsuffix cgo -o handler . && \
    go test $(go list ./... | grep -v /vendor/) -cover

FROM alpine:3.11.6
# Add non root user and certs
RUN apk --no-cache add ca-certificates \
    && addgroup -S app && adduser -S -g app app \
    && mkdir -p /home/app \
    && chown app /home/app

WORKDIR /home/app

COPY --from=build /go/src/handler/handler    .
COPY --from=build /usr/bin/fwatchdog         .
COPY --from=build /go/src/handler/function/  .

RUN chown -R app /home/app

USER app

ENV fprocess="./handler"
ENV mode="http"
ENV upstream_url="http://127.0.0.1:8082"

ENV read_timeout: 120
ENV write_timeout: 120
ENV exec_timeout: 0
ENV write_debug: true
ENV combine_output: false

CMD ["./fwatchdog"]
