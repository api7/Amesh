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
	"context"
	"github.com/api7/amesh/pkg/amesh/provisioner"
	"github.com/api7/gopkg/pkg/log"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net"
	"os"
	"strings"
	"time"
	"unsafe"

	"github.com/api7/amesh/pkg/apisix"
)

type Storage interface {
	Store(string, string)
}

type Agent struct {
	ctx       context.Context
	version   int64
	xdsSource string
	logger    *log.Logger

	provisioner provisioner.Provisioner

	TargetStorage Storage
}

func getNamespace() string {
	namespace := "default"
	if value := os.Getenv("POD_NAMESPACE"); value != "" {
		namespace = value
	}
	return namespace
}

func getIpAddr() (string, error) {
	var (
		ipAddr string
	)
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, iface := range ifaces {
		if iface.Name != "lo" {
			addrs, err := iface.Addrs()
			if err != nil {
				return "", err
			}
			if len(addrs) > 0 {
				ipAddr = strings.Split(addrs[0].String(), "/")[0]
			}
		}
	}
	if ipAddr == "" {
		ipAddr = "127.0.0.1"
	}
	return ipAddr, nil
}

func NewAgent(ctx context.Context, src string, zone unsafe.Pointer, logLevel, logOutput string) (*Agent, error) {
	ipAddr, err := getIpAddr()
	if err != nil {
		return nil, err
	}

	p, err := provisioner.NewXDSProvisioner(&provisioner.Config{
		RunId:           uuid.NewString(),
		LogLevel:        logLevel,
		LogOutput:       logOutput,
		XDSConfigSource: src,
		Namespace:       getNamespace(),
		IpAddress:       ipAddr,
	})
	if err != nil {
		return nil, err
	}

	logger, err := log.NewLogger(
		log.WithContext("sidecar"),
		log.WithLogLevel(logLevel),
		log.WithOutputFile(logOutput),
	)
	if err != nil {
		return nil, err
	}

	return &Agent{
		ctx:           ctx,
		version:       time.Now().Unix(),
		xdsSource:     src,
		logger:        logger,
		provisioner:   p,
		TargetStorage: apisix.NewSharedDictStorage(zone),
	}, nil
}

func (g *Agent) Stop() {
}

func (g *Agent) Run(stop <-chan struct{}) error {
	g.logger.Infow("sidecar started")
	defer g.logger.Info("sidecar exited")

	go func() {
		if err := g.provisioner.Run(stop); err != nil {
			g.logger.Fatalw("provisioner run failed",
				zap.Error(err),
			)
		}
	}()

loop:
	for {
		events, ok := <-g.provisioner.Channel()
		if !ok {
			break loop
		}
		g.storeEvents(events)
	}

	return nil
}

func (g *Agent) storeEvents(events []provisioner.Event) {

}
