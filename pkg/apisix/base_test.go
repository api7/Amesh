package apisix

import (
	"testing"

	"github.com/api7/gopkg/pkg/log"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestVarToComparableString(t *testing.T) {
	cases := []struct {
		a      *Var
		b      *Var
		result bool
	}{
		{
			a:      &Var{},
			b:      &Var{},
			result: true,
		},
		{
			a:      &Var{"A", "==", "B"},
			b:      &Var{"A", "==", "B"},
			result: true,
		},
		{
			a:      &Var{"A", "in", &Var{"B", "C"}},
			b:      &Var{"A", "in", &Var{"B", "C"}},
			result: true,
		},
		{
			a:      &Var{"A", "in", &Var{"B", "C"}},
			b:      &Var{"A", "in", &Var{"BC"}},
			result: false,
		},
		{
			a:      &Var{"A", "in", &Var{"B", "C"}},
			b:      &Var{"A", "in", &Var{"B.C"}},
			result: false,
		},
		{
			a:      &Var{"A", "in", &Var{"B.", "C"}},
			b:      &Var{"A", "in", &Var{"B..C"}},
			result: false,
		},
	}

	for _, tc := range cases {
		aStr := tc.a.ToComparableString()
		bStr := tc.b.ToComparableString()
		log.Errorw("compare",
			zap.String("a", aStr),
			zap.String("b", bStr),
		)
		assert.Equal(t, tc.result, aStr == bStr)
	}
}
