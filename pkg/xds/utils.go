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
package xds

import "strings"

// GenNodeId generates an id used for xDS protocol. The format is like:
// sidecar~172.10.0.2~12345asad034~default.svc.cluster.local
func GenNodeId(runId, ipAddr, dnsDomain string) string {
	var buf strings.Builder
	buf.WriteString("sidecar~")
	buf.WriteString(ipAddr)
	buf.WriteString("~")
	buf.WriteString(runId)
	buf.WriteString("~")
	buf.WriteString(dnsDomain)
	return buf.String()
}
