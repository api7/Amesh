// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestLabelSelectorCompare(t *testing.T) {
	assert.True(t, LabelSelectorEqual(labels.Everything(), labels.Everything()))

	RunCase := func(expect bool, a, b *metav1.LabelSelector) {
		if expect {
			return
		}

		ToSelector := func(metaSelector *metav1.LabelSelector) (selector labels.Selector, err error) {
			if metaSelector == nil || (len(metaSelector.MatchLabels)+len(metaSelector.MatchExpressions) == 0) {
				selector = labels.Everything()
			} else {
				selector, err = metav1.LabelSelectorAsSelector(metaSelector)
				if err != nil {
					return nil, err
				}
			}
			return
		}

		selectorA, err := ToSelector(a)
		assert.Nil(t, err)
		selectorB, err := ToSelector(b)
		assert.Nil(t, err)

		assert.Equal(t, expect, LabelSelectorEqual(selectorA, selectorB))
	}

	// --- MatchLabels ---
	RunCase(true, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label": "value",
		},
	}, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label": "value",
		},
	})

	RunCase(true, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label-a": "value-a",
			"label-b": "value-b",
			"label-c": "value-c",
		},
	}, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label-b": "value-b",
			"label-c": "value-c",
			"label-a": "value-a",
		},
	})

	RunCase(false, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label": "value",
		},
	}, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label": "value-changed",
		},
	})

	// --- MatchExpressions ---
	RunCase(true, &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "label", Operator: metav1.LabelSelectorOpIn, Values: []string{"a", "b"}},
		},
	}, &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "label", Operator: metav1.LabelSelectorOpIn, Values: []string{"b", "a"}},
		},
	})

	RunCase(false, &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "label", Operator: metav1.LabelSelectorOpIn, Values: []string{"a", "b"}},
		},
	}, &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "label", Operator: metav1.LabelSelectorOpIn, Values: []string{"b"}},
		},
	})

	// -- Both ---
	RunCase(true, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label-a": "value-a",
			"label-b": "value-b",
			"label-c": "value-c",
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "label", Operator: metav1.LabelSelectorOpIn, Values: []string{"a", "b"}},
		},
	}, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label-b": "value-b",
			"label-c": "value-c",
			"label-a": "value-a",
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "label", Operator: metav1.LabelSelectorOpIn, Values: []string{"b", "a"}},
		},
	})

	RunCase(false, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label": "value",
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "label", Operator: metav1.LabelSelectorOpIn, Values: []string{"a", "b"}},
		},
	}, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label": "value-changed",
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "label", Operator: metav1.LabelSelectorOpIn, Values: []string{"b", "a"}},
		},
	})
	RunCase(false, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label": "value",
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "label", Operator: metav1.LabelSelectorOpIn, Values: []string{"a", "b"}},
		},
	}, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label": "value",
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "label", Operator: metav1.LabelSelectorOpIn, Values: []string{"a"}},
		},
	})
}
