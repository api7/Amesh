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
package amesh

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/api7/amesh/pkg/apisix"
)

func (g *Agent) RunDemo() error {
	for {
		select {
		case <-g.ctx.Done():
			return nil
		case <-time.After(time.Second * time.Duration(rand.Intn(10))):
			g.version = time.Now().Unix()

			routes := g.generateRandomRoutes()
			upstreams := g.generateRandomUpstreams()

			routesJson, err := json.Marshal(routes)
			if err != nil {
				return err
			}
			upstreamsJson, err := json.Marshal(upstreams)
			if err != nil {
				return err
			}

			g.DataStorage.Store("routes", string(routesJson))
			g.DataStorage.Store("upstreams", string(upstreamsJson))
		}
	}
	return nil
}

func (g *Agent) generateRandomRoutes() []*apisix.Route {
	return []*apisix.Route{
		{
			Uris:        nil,
			Name:        fmt.Sprintf("route-%v", g.version),
			Id:          "1",
			Desc:        fmt.Sprintf("Route at %v", g.version),
			Priority:    0,
			Methods:     []string{"GET", "POST"},
			Hosts:       []string{"example.com"},
			RemoteAddrs: []string{"127.0.0.1"},
			Vars:        nil,
			Plugins:     nil,
			UpstreamId:  "1",
			Status:      apisix.RouteEnable,
			Labels:      nil,
			CreateTime:  time.Now().Unix(),
			UpdateTime:  time.Now().Unix(),
		},
	}
}

func (g *Agent) generateRandomUpstreams() []*apisix.Upstream {
	return []*apisix.Upstream{
		{
			CreateTime: time.Now().Unix(),
			UpdateTime: time.Now().Unix(),
			Nodes: []*apisix.Node{
				{
					Host: "127.0.0.1",
					Port: 1980,
				},
			},
			Type:   apisix.LoadBalanceTypeRoundrobin,
			Scheme: "",
			Labels: nil,
			Name:   fmt.Sprintf("upstream-%v", g.version),
			Desc:   fmt.Sprintf("Upstream at %v", g.version),
			Id:     "1",
		},
	}
}
