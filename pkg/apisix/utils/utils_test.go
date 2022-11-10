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
