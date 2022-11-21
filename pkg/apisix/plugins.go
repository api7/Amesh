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

type FaultInjectionAbort struct {
	HttpStatus uint32 `json:"http_status,omitempty"`
	Body       string `json:"body,omitempty"`
	Percentage uint32 `json:"percentage,omitempty"`
	Vars       []*Var `json:"vars,omitempty"`
}

type FaultInjectionDelay struct {
	// Duration in seconds
	Duration   int64  `json:"duration,omitempty"`
	Percentage uint32 `json:"percentage,omitempty"`
	Vars       []*Var `json:"vars,omitempty" json:"vars,omitempty"`
}

type FaultInjection struct {
	Abort *FaultInjectionAbort `json:"abort,omitempty"`
	Delay *FaultInjectionDelay `json:"delay,omitempty"`
}
