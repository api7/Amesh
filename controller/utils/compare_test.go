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
