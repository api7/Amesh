// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenNodeId(t *testing.T) {
	id := GenNodeId("12345", "10.0.5.3", "default.svc.cluster.local")
	assert.Equal(t, "sidecar~10.0.5.3~12345~default.svc.cluster.local", id)
}
