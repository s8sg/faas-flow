module github.com/s8sg/faas-flow

replace github.com/s8sg/faas-flow => ./

go 1.14

require (
	github.com/alexellis/hmac v0.0.0-20180624211220-5c52ab81c0de
	github.com/boltdb/bolt v1.3.1
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rs/xid v1.2.1
	github.com/s8sg/faasflow v0.5.0
	github.com/stretchr/testify v1.5.1 // indirect
	github.com/uber/jaeger-client-go v2.23.1+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	go.uber.org/atomic v1.6.0 // indirect
	golang.org/x/sys v0.0.0-20200523222454-059865788121 // indirect
)
