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

import "sort"

// StringSet represents a set which elements are string.
type StringSet map[string]struct{}

// Add adds an element to set.
func (set StringSet) Add(e string) {
	set[e] = struct{}{}
}

// Equal compares two string set and checks whether they are identical.
func (set StringSet) Equal(set2 StringSet) bool {
	if len(set) != len(set2) {
		return false
	}
	for e := range set2 {
		if _, ok := set[e]; !ok {
			return false
		}
	}
	for e := range set {
		if _, ok := set2[e]; !ok {
			return false
		}
	}
	return true
}

// Strings converts the string set to a string slice.
func (set StringSet) Strings() []string {
	s := make([]string, 0, len(set))
	for e := range set {
		s = append(s, e)
	}
	return s
}

// OrderedStrings converts the string set to a sorted string slice.
func (set StringSet) OrderedStrings() []string {
	s := set.Strings()
	sort.Strings(s)
	return s
}
