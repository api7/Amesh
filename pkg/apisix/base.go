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
package apisix

import "strings"

// Valid Var array:
// ["XXX", "==", "YYY"]
// ["XXX", "in", ["A", "B", "C"]]
// A Var should be string or string array
type Var []interface{}

// ToComparableString converts Var to string
// The function doesn't guarantee that there are no conflicts in any case.
func (v *Var) ToComparableString() string {
	s := ""

	for _, val := range *v {
		switch value := val.(type) {
		case string:
			// escape dot to avoid conflict
			s += strings.Replace(value, ".", `..`, -1) + "."
		case *Var:
			s += value.ToComparableString() + "."
		}
	}

	return s
}

// Timeout represents the timeout settings.
// It's worth to note that the timeout is used to control the time duration
// between two successive I/O operations. It doesn't constraint the whole I/O
// operation durations.
type Timeout struct {
	// connect controls the connect timeout in seconds.
	Connect float64 `json:"connect,omitempty"`
	// send controls the send timeout in seconds.
	Send float64 `json:"send,omitempty"`
	// read controls the read timeout in seconds.
	Read float64 `json:"read,omitempty"`
}
