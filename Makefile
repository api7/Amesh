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
REGISTRY ?="localhost:5000"
ISTIO_RELEASE ?= 1.13.1
REQUIRE_REBUILD_IPTABLES_IMAGE ?= true
REQUIRE_REBUILD_CONTROLLER_IMAGE ?= true

export PREBUILT_AMESH_APISIX_IMAGE ?= api7/amesh-apisix:v0.0.3
export AMESH_STANDALONE_IMAGE ?= amesh-standalone
export AMESH_STANDALONE_IMAGE_TAG ?= dev
export AMESH_SIDECAR_IMAGE ?= amesh-sidecar
export AMESH_SIDECAR_IMAGE_TAG ?= dev
export AMESH_IPTABLES_IMAGE ?= amesh-iptables
export AMESH_IPTABLES_IMAGE_TAG ?= dev
export AMESH_SO_IMAGE ?= amesh-so
export AMESH_SO_IMAGE_TAG ?= dev

# vars for `make install`
OPENRESTY_PREFIX ?= /usr/local/openresty
INSTALL ?= install

.PHONY: create-bin-dir
create-bin-dir:
	@mkdir -p $(BINDIR)

.PHONY: build-amesh-standalone
build-amesh-standalone: create-bin-dir
	CGO_ENABLED=0 go build -o $(BINDIR)/amesh-standalone -tags standalone ./cmd/standalone

.PHONY: build-amesh-so
build-amesh-so: create-bin-dir
	go build -o $(BINDIR)/libxds.so -buildmode=c-shared -gcflags=-shared -asmflags=-shared -installsuffix=_shared -a  -tags shared_lib ./cmd/dynamic

.PHONY: build-amesh-iptables-image
build-amesh-iptables-image:
ifeq ($(REQUIRE_REBUILD_IPTABLES_IMAGE), true)
ifeq ($(ENABLE_PROXY), true)
	docker build -f Dockerfiles/iptables.Dockerfile --build-arg ENABLE_PROXY=true -t $(AMESH_IPTABLES_IMAGE):$(AMESH_IPTABLES_IMAGE_TAG) .
else
	docker build -f Dockerfiles/iptables.Dockerfile -t $(AMESH_IPTABLES_IMAGE):$(AMESH_IPTABLES_IMAGE_TAG) .
endif
else
	# reuse pre-built image
	docker pull api7/amesh-iptables:v0.0.2
	docker tag api7/amesh-iptables:v0.0.2 $(AMESH_IPTABLES_IMAGE):$(AMESH_IPTABLES_IMAGE_TAG)
endif

.PHONY: build-amesh-standalone-image
build-amesh-standalone-image:
ifeq ($(ENABLE_PROXY), true)
	docker build -f Dockerfiles/standalone.Dockerfile --build-arg ENABLE_PROXY=true -t $(AMESH_STANDALONE_IMAGE):$(AMESH_STANDALONE_IMAGE_TAG) .
else
	docker build -f Dockerfiles/standalone.Dockerfile -t $(AMESH_STANDALONE_IMAGE):$(AMESH_STANDALONE_IMAGE_TAG) .
endif

.PHONY: build-amesh-so-image
build-amesh-so-image:
ifeq ($(ENABLE_PROXY), true)
	docker build -f Dockerfiles/luajit.Dockerfile --build-arg ENABLE_PROXY=true -t $(AMESH_SO_IMAGE):$(AMESH_SO_IMAGE_TAG) .
else
	docker build -f Dockerfiles/luajit.Dockerfile -t $(AMESH_SO_IMAGE):$(AMESH_SO_IMAGE_TAG) .
endif

.PHONY: build-apisix-image
build-apisix-image:
	docker build \
		-f Dockerfiles/apisix.Dockerfile \
		-t amesh-apisix:dev \
		--build-arg ENABLE_PROXY=true \
		--build-arg LUAROCKS_SERVER=https://luarocks.cn .

.PHONY: build-amesh-sidecar-image
build-amesh-sidecar-image:
	docker pull $(PREBUILT_AMESH_APISIX_IMAGE)
	docker tag $(PREBUILT_AMESH_APISIX_IMAGE) amesh-apisix:dev
	docker build --build-arg ENABLE_PROXY=$(ENABLE_PROXY) \
		-f Dockerfiles/amesh-sidecar.Dockerfile \
		-t $(AMESH_SIDECAR_IMAGE):$(AMESH_SIDECAR_IMAGE_TAG) .

.PHONY: build-amesh-controller-image
build-amesh-controller-image:
ifeq ($(REQUIRE_REBUILD_CONTROLLER_IMAGE), true)
	cd controller && \
	make docker-build
else
	cd controller && \
	make docker-build
#	# reuse pre-built image
#	docker pull api7/amesh-controller:latest
endif
	docker tag amesh-controller:latest $(REGISTRY)/amesh-controller:latest

.PHONY: prepare-images
prepare-images: build-amesh-iptables-image build-amesh-controller-image build-amesh-sidecar-image
	docker tag $(AMESH_IPTABLES_IMAGE):$(AMESH_IPTABLES_IMAGE_TAG) $(REGISTRY)/$(AMESH_IPTABLES_IMAGE):$(AMESH_IPTABLES_IMAGE_TAG)
	docker tag $(AMESH_SIDECAR_IMAGE):$(AMESH_SIDECAR_IMAGE_TAG) $(REGISTRY)/$(AMESH_SIDECAR_IMAGE):$(AMESH_SIDECAR_IMAGE_TAG)

	docker pull istio/pilot:1.13.1
	docker tag istio/pilot:1.13.1 $(REGISTRY)/istio/pilot:1.13.1

	docker pull nginx:1.19.3
	docker tag nginx:1.19.3 $(REGISTRY)/nginx:1.19.3

	docker pull kennethreitz/httpbin
	docker tag kennethreitz/httpbin $(REGISTRY)/kennethreitz/httpbin

	docker pull curlimages/curl
	docker tag curlimages/curl $(REGISTRY)/curlimages/curl

.PHONY: push-images
push-images:
	docker push $(REGISTRY)/$(AMESH_IPTABLES_IMAGE):$(AMESH_IPTABLES_IMAGE_TAG)
	docker push $(REGISTRY)/$(AMESH_SIDECAR_IMAGE):$(AMESH_SIDECAR_IMAGE_TAG)
	docker push $(REGISTRY)/amesh-controller:latest

	docker push $(REGISTRY)/istio/pilot:1.13.1
	docker push $(REGISTRY)/nginx:1.19.3
	docker push $(REGISTRY)/kennethreitz/httpbin
	docker push $(REGISTRY)/curlimages/curl

.PHONY: kind-up
kind-up:
	./scripts/kind-with-registry.sh

.PHONY: e2e-test
e2e-test:
	kubectl apply -k $(shell pwd)/controller/config/crd
	AMESH_E2E_HOME=$(shell pwd)/e2e \
		cd e2e && \
		go env -w GOFLAGS="-mod=mod" && \
		ginkgo -cover -coverprofile=coverage.txt -r --randomize-suites --randomize-all --trace -p --nodes=$(E2E_CONCURRENCY)

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

.PHONY install:
install: build-amesh-so
	$(INSTALL) -m 664 $(BINDIR)/libxds.so $(OPENRESTY_PREFIX)/

.PHONY install-amesh:
install-amesh:
	helm install amesh --create-namespace \
	 --namespace istio-system \
	 --set pilot.image=istio/pilot:$(ISTIO_RELEASE) \
	 --set global.proxy.privileged=true \
	 --set global.proxy_init.image=$(REGISTRY)/amesh-iptables:dev \
	 --set global.proxy.image=$(REGISTRY)/amesh-sidecar:dev \
	 --set global.imagePullPolicy=IfNotPresent \
	 --set global.hub="docker.io/istio" \
	 --set global.tag="$(ISTIO_RELEASE)" \
	 ./charts/amesh
