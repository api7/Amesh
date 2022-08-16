FROM golang:1.16.5 as amesh-sidecar-build-stage

ARG ENABLE_PROXY=false
WORKDIR /amesh

COPY go.* ./
RUN if [ "$ENABLE_PROXY" = "true" ]; then go env -w GOPROXY=https://goproxy.cn,direct ; fi \
    && go mod download

COPY Makefile Makefile
COPY cmd/dynamic cmd/dynamic
COPY pkg/ pkg/
RUN if [ "$ENABLE_PROXY" = "true" ]; then go env -w GOPROXY=https://goproxy.cn,direct ; fi \
    && make build-amesh-so

FROM amesh-apisix:dev

WORKDIR /usr/local/apisix
COPY Dockerfiles/config.yaml conf/config.yaml
COPY --from=amesh-sidecar-build-stage /amesh/bin/libxds.so libxds.so
