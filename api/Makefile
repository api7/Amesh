# Copyright 2021 The Amesh Authors
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

PROTOC_GEN_VALIDATE_DIR ?= ""
GOOGLE_PROTOBUF_DIR ?= ""
GOOGLEAPIS_DIR ?= ""
GO_OUT_DIR ?= go
GO_TMP_DIR ?= .
DOCS_OUT_DIR ?= docs

PKG="github.com/api7/amesh/api"

.PHONY: go
go: go_proto
	rm -rf $(GO_TMP_DIR)/github.com

.PHONY: go_proto
go_proto:
	mkdir -p $(GO_OUT_DIR)/proto/v1
	protoc -I . \
		-I $(PROTOC_GEN_VALIDATE_DIR) \
		-I $(GOOGLE_PROTOBUF_DIR) \
		-I $(GOOGLEAPIS_DIR) \
		--go-grpc_out=$(GO_TMP_DIR) \
		--go_out=$(GO_TMP_DIR) \
		--validate_out="lang=go:$(GO_TMP_DIR)" \
		proto/v1/*.proto
	cp -r $(GO_TMP_DIR)/$(PKG)/proto/v1 $(GO_OUT_DIR)/proto

.PHONY: docs
docs:
	protoc -I . \
		-I $(PROTOC_GEN_VALIDATE_DIR) \
		-I $(GOOGLE_PROTOBUF_DIR) \
		-I $(GOOGLEAPIS_DIR) \
		--doc_out=$(DOCS_OUT_DIR) \
		--doc_opt=markdown,proto.md \
		proto/v1/*.proto
