# Copyright 2022 The Amesh Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.18 as amesh-sidecar-build-stage

ARG ENABLE_PROXY=false
WORKDIR /amesh

COPY api/ api/
COPY controller/go.* controller/
COPY controller/apis controller/apis
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
