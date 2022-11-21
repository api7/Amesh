// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/pkg/apisix"
)

func TestIsSameNodes(t *testing.T) {
	type testCase struct {
		nodesA []*apisix.Node
		nodesB []*apisix.Node
		equal  bool
	}

	cases := []testCase{
		{
			nodesA: []*apisix.Node{},
			nodesB: []*apisix.Node{},
			equal:  true,
		},
		{
			nodesA: []*apisix.Node{
				{
					Host:     "10.0.0.20",
					Port:     80,
					Weight:   1,
					Metadata: nil,
				},
			},
			nodesB: []*apisix.Node{},
			equal:  false,
		},
		{
			nodesA: []*apisix.Node{
				{
					Host:     "10.0.0.20",
					Port:     80,
					Weight:   1,
					Metadata: nil,
				},
			},
			nodesB: []*apisix.Node{
				{
					Host:     "10.0.0.20",
					Port:     80,
					Weight:   1,
					Metadata: nil,
				},
			},
			equal: true,
		},
		{
			nodesA: []*apisix.Node{
				{
					Host:     "10.0.0.20",
					Port:     80,
					Weight:   1,
					Metadata: nil,
				},
				{
					Host:     "10.0.0.21",
					Port:     80,
					Weight:   1,
					Metadata: nil,
				},
			},
			nodesB: []*apisix.Node{
				{
					Host:     "10.0.0.21",
					Port:     80,
					Weight:   1,
					Metadata: nil,
				},
				{
					Host:     "10.0.0.20",
					Port:     80,
					Weight:   1,
					Metadata: nil,
				},
			},
			equal: true,
		},
		{
			nodesA: []*apisix.Node{
				{
					Host:     "10.0.0.20",
					Port:     8080,
					Weight:   1,
					Metadata: nil,
				},
				{
					Host:     "10.0.0.21",
					Port:     80,
					Weight:   1,
					Metadata: nil,
				},
			},
			nodesB: []*apisix.Node{
				{
					Host:     "10.0.0.21",
					Port:     80,
					Weight:   1,
					Metadata: nil,
				},
				{
					Host:     "10.0.0.20",
					Port:     80,
					Weight:   1,
					Metadata: nil,
				},
			},
			equal: false,
		},
	}

	for _, testCase := range cases {
		assert.Equal(t, testCase.equal, IsSameNodes(testCase.nodesA, testCase.nodesB))
		assert.Equal(t, testCase.equal, IsSameNodes(testCase.nodesB, testCase.nodesA))
	}
}
