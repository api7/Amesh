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

package framework

import (
	"github.com/api7/gopkg/pkg/log"

	"github.com/api7/amesh/e2e/framework/utils"
)

const (
	virtualServiceWithFaultAbort = `
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: {{ .Name }}
spec:
  hosts:
  - {{ .Host }}
  http:
  - match:
    - headers:
        {{ .Header }}:
          exact: {{ .HeaderValue }}
    fault:
      abort:
        httpStatus: {{ .Status }}
        percentage:
          value: {{ .Percentage }}
    route:
    - destination:
        host: {{ .Host }}
  - route:
    - destination:
        host: {{ .Host }}
`
)

type FaultAbortArgs struct {
	Name string
	Host string

	Header      string
	HeaderValue string

	Status     int
	Percentage int
}

func (args *FaultAbortArgs) Normalize() {
	if args.Status == 0 {
		args.Status = 555
	}
	if args.Percentage == 0 {
		args.Percentage = 100
	}
}

func (f *Framework) CreateVirtualServiceWithFaultAbort(args *FaultAbortArgs) {
	args.Normalize()

	artifact, err := utils.RenderManifest(virtualServiceWithFaultAbort, args)
	utils.AssertNil(err, "render virtual service fault abort template")
	err = f.ApplyResourceFromString(artifact)
	if err != nil {
		log.Errorf("failed to apply virtual service fault abort: %s", err.Error())
	}
	utils.AssertNil(err, "apply virtual service fault abort")
}
