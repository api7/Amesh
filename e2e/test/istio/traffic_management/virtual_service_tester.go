package traffic_management

import (
	"github.com/api7/amesh/e2e/framework"
)

type VirtualServiceTester struct {
	Config *framework.VirtualServiceConfig
}

func (tester *VirtualServiceTester) SetRouteDestination(index int, name string, dest ...string) {

}
