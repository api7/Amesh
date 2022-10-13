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
package version

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"time"
)

var (
	// The following fields are populated at build time using -ldflags -X.
	_version     = "unknown"
	_gitRevision = "unknown"
	_timestamp   = "0"
)

// Short returns a short version representation.
func Short() string {
	return _version
}

// String returns a readable version info.
func String() string {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "Version: %s\n", _version)
	fmt.Fprintf(buf, "Git SHA: %s\n", _gitRevision)
	fmt.Fprintf(buf, "Go Version: %s\n", runtime.Version())
	fmt.Fprintf(buf, "OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	ts, err := strconv.ParseInt(_timestamp, 10, 32)
	if err != nil {
		fmt.Fprintln(buf, "Build Date: unknown")
	} else {
		date := time.Unix(ts, 0)
		fmt.Fprintf(buf, "Build Date: %s\n", date.String())
	}

	return buf.String()
}
