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
package provisioner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/api7/gopkg/pkg/log"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/api7/amesh/pkg/version"
	"github.com/api7/amesh/pkg/xds"
)

type xdsProvisioner struct {
	src    string
	node   *corev3.Node
	logger *log.Logger

	evChan  chan []Event
	resetCh chan error
}

type Config struct {
	// Running Id of this instance, it will be filled by
	// a random string when the instance started.
	RunId string
	// The minimum log level that will be printed.
	LogLevel string `json:"log_level" yaml:"log_level"`
	// The destination of logs.
	LogOutput string `json:"log_output" yaml:"log_output"`
	// The xds source
	XDSConfigSource string `json:"xds_config_source" yaml:"xds_config_source"`

	Namespace string
	IpAddress string
}

func NewXDSProvisioner(cfg *Config) (Provisioner, error) {
	logger, err := log.NewLogger(
		log.WithOutputFile(cfg.LogOutput),
		log.WithLogLevel(cfg.LogLevel),
		log.WithContext("xds-grpc-provisioner"),
	)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(cfg.XDSConfigSource, "grpc://") {
		return nil, errors.New("bad xds config source")
	}
	src := strings.TrimPrefix(cfg.XDSConfigSource, "grpc://")

	// TODO make domain suffix configurable
	dnsDomain := cfg.Namespace + ".svc.cluster.local"
	node := &corev3.Node{
		Id:            xds.GenNodeId(cfg.RunId, cfg.IpAddress, dnsDomain),
		UserAgentName: fmt.Sprintf("amesh/%s", version.Short()),
	}

	return &xdsProvisioner{
		src:     src,
		node:    node,
		logger:  logger,
		evChan:  make(chan []Event),
		resetCh: make(chan error),
	}, nil
}

func (p *xdsProvisioner) Channel() <-chan []Event {
	return p.evChan
}

func (p *xdsProvisioner) Run(stop <-chan struct{}) error {
	p.logger.Infow("provisioner started")
	defer p.logger.Info("provisioner exited")
	defer close(p.evChan)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		conn, err := grpc.DialContext(ctx, p.src,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			cancel()
			p.logger.Errorw("failed to conn xds source",
				zap.Error(err),
				zap.String("xds_source", p.src),
			)
			return err
		}
		cleanup := func() {
			cancel()
			if err := conn.Close(); err != nil {
				p.logger.Errorw("failed to close gRPC connection to XDS config source",
					zap.Error(err),
					zap.String("src", p.src),
				)
			}
		}
		p.logger.Info("xds connected")

		if err := p.run(ctx, conn); err != nil {
			cleanup()
			return err
		}

		select {
		case <-stop:
			cleanup()
			return nil
		case err = <-p.resetCh:
			cleanup()
			p.logger.Errorw("grpc client reset, try reconnect",
				zap.Error(err),
			)
			continue
		}
	}
}

func (p *xdsProvisioner) run(ctx context.Context, conn *grpc.ClientConn) error {
	client, err := discoveryv3.NewAggregatedDiscoveryServiceClient(conn).StreamAggregatedResources(ctx)
	if err != nil {
		return err
	}

	// TODO
	_ = client
	return nil
}
