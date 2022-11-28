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
