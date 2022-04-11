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
//
package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringSet(t *testing.T) {
	s := StringSet{}
	s.Add("123")
	s.Add("456")
	s2 := StringSet{}
	s2.Add("123")
	s2.Add("456")

	assert.Equal(t, s.Equals(s2), true)
	s2.Add("111")
	assert.Equal(t, s.Equals(s2), false)
}

func TestStringSetToArray(t *testing.T) {
	s := StringSet{}
	s.Add("123")
	s.Add("456")
	s2 := StringSet{}
	s2.Add("456")
	s2.Add("123")

	assert.NotNil(t, s.OrderedStrings())
	assert.NotNil(t, s2.OrderedStrings())
	assert.Equal(t, s.OrderedStrings(), s2.OrderedStrings())
}
