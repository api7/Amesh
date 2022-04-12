# Copyright 2022 The Amesh Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

BINDIR ?= ./bin
ENABLE_PROXY ?= false
export AMESH_SIDECAR_IMAGE ?= amesh-sidecar
export AMESH_SIDECAR_IMAGE_TAG ?= dev
export AMESH_IPTABLES_IMAGE ?= amesh-iptables
export AMESH_IPTABLES_IMAGE_TAG ?= dev
export AMESH_SO_IMAGE ?= amesh-so
export AMESH_SO_IMAGE_TAG ?= dev

.PHONY: create-bin-dir
create-bin-dir:
	@mkdir -p $(BINDIR)

.PHONY: build-amesh-sidecar
build-amesh-sidecar: create-bin-dir
	CGO_ENABLED=0 go build -o $(BINDIR)/amesh-sidecar -tags sidecar ./cmd/sidecar

.PHONY: build-amesh-so
build-amesh-so: create-bin-dir
	go build -o $(BINDIR)/amesh.so -buildmode=c-shared -gcflags=-shared -asmflags=-shared -installsuffix=_shared -a  -tags shared_lib ./cmd/dynamic

.PHONY: build-amesh-iptables-image
build-amesh-iptables-image:
ifeq ($(ENABLE_PROXY), true)
	@docker build -f Dockerfiles/iptables.Dockerfile --build-arg ENABLE_PROXY=true -t $(AMESH_IPTABLES_IMAGE):$(AMESH_IPTABLES_IMAGE_TAG) .
else
	@docker build -f Dockerfiles/iptables.Dockerfile -t $(AMESH_IPTABLES_IMAGE):$(AMESH_IPTABLES_IMAGE_TAG) .
endif

.PHONY: build-amesh-sidecar-image
build-amesh-sidecar-image:
ifeq ($(ENABLE_PROXY), true)
	@docker build -f Dockerfiles/sidecar.Dockerfile --build-arg ENABLE_PROXY=true -t $(AMESH_SIDECAR_IMAGE):$(AMESH_SIDECAR_IMAGE_TAG) .
else
	@docker build -f Dockerfiles/sidecar.Dockerfile -t $(AMESH_SIDECAR_IMAGE):$(AMESH_SIDECAR_IMAGE_TAG) .
endif

.PHONY: build-amesh-so-image
build-amesh-so-image:
ifeq ($(ENABLE_PROXY), true)
	@docker build -f Dockerfiles/luajit.Dockerfile --build-arg ENABLE_PROXY=true -t $(AMESH_SO_IMAGE):$(AMESH_SO_IMAGE_TAG) .
else
	@docker build -f Dockerfiles/luajit.Dockerfile -t $(AMESH_SO_IMAGE):$(AMESH_SO_IMAGE_TAG) .
endif

.PHONY: build-apisix-image
build-apisix-image:
	docker build \
		-f Dockerfiles/apisix.Dockerfile \
		-t amesh-apisix:dev \
		--build-arg ENABLE_PROXY=true \
		--build-arg LUAROCKS_SERVER=https://luarocks.cn .

.PHONY verify-license:
verify-license:
	@docker pull apache/skywalking-eyes
	docker run -it --rm -v $(PWD):/github/workspace apache/skywalking-eyes header check

.PHONY update-license:
update-license:
	@docker pull apache/skywalking-eyes
	docker run -it --rm -v $(PWD):/github/workspace apache/skywalking-eyes header fix

.PHONY update-imports:
update-imports:
	bash -c 'go install github.com/incu6us/goimports-reviser/v2@latest'
	./scripts/update-imports.sh
