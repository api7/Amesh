FROM golang:1.16.5 as amesh-sidecar-build-stage

ARG ENABLE_PROXY=false
WORKDIR /amesh

COPY go.* ./
RUN if [ "$ENABLE_PROXY" = "true" ]; then go env -w GOPROXY=https://goproxy.cn,direct ; fi \
    && go mod download

COPY Makefile Makefile
COPY cmd/ cmd/
COPY pkg/ pkg/
RUN if [ "$ENABLE_PROXY" = "true" ]; then go env -w GOPROXY=https://goproxy.cn,direct ; fi \
    && make build-amesh-sidecar

FROM centos:7

WORKDIR /usr/local/amesh

COPY --from=amesh-sidecar-build-stage /amesh/bin/amesh-sidecar ./

#COPY ./bin/amesh-sidecar .

ENTRYPOINT ["/usr/local/amesh/amesh-sidecar"]
