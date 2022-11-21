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
