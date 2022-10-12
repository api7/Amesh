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
package amesh

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/api7/gopkg/pkg/log"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/api7/amesh/pkg/amesh/provisioner"
	"github.com/api7/amesh/pkg/amesh/types"
	"github.com/api7/amesh/pkg/apisix"
	"github.com/api7/amesh/pkg/apisix/storage"
)

type Agent struct {
	ctx       context.Context
	version   int64
	xdsSource string
	logger    *log.Logger

	provisioner types.Provisioner

	DataStorage    apisix.Storage
	VersionStorage apisix.Storage
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

func NewAgent(ctx context.Context, src, ameshGrpc string, dataZone, versionZone unsafe.Pointer, logLevel, logOutput string) (*Agent, error) {
	ipAddr, err := getIpAddr()
	if err != nil {
		return nil, err
	}

	p, err := provisioner.NewXDSProvisioner(&provisioner.Config{
		RunId:             uuid.NewString(),
		LogLevel:          logLevel,
		LogOutput:         logOutput,
		XDSConfigSource:   src,
		AmeshConfigSource: ameshGrpc,
		Namespace:         getNamespace(),
		IpAddress:         ipAddr,
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
		ctx:            ctx,
		version:        time.Now().Unix(),
		xdsSource:      src,
		logger:         logger,
		provisioner:    p,
		DataStorage:    storage.NewSharedDictStorage(dataZone),
		VersionStorage: storage.NewSharedDictStorage(versionZone),
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

	go func() {
		portStr := os.Getenv("STATUS_SERVER_PORT")
		if portStr == "" {
			// Do not start status server
			g.logger.Infof("STATUS_SERVER_PORT is empty, status server won't start")
			return
		}
		port := 9999
		parsed, err := strconv.ParseInt(portStr, 10, 32)
		if err != nil {
			g.logger.Fatalw("failed to parse status server port, using default",
				zap.Error(err),
				zap.String("value", portStr),
			)
		} else {
			port = int(parsed)
		}
		http.HandleFunc("/status", func(writer http.ResponseWriter, request *http.Request) {
			status, err := g.provisioner.Status()
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				_, writeErr := writer.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
				if writeErr != nil {
					g.logger.Fatalw("failed to write status",
						zap.Error(writeErr),
					)
				}
				return
			}
			writer.WriteHeader(http.StatusOK)
			_, err = writer.Write([]byte(status))
			if err != nil {
				g.logger.Fatalw("failed to write status",
					zap.Error(err),
				)
			}
			return
		})
		err = http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
		if err != nil {
			g.logger.Fatalw("failed to run status server",
				zap.Error(err),
			)
			return
		}
	}()

loop:
	for {
		select {
		case <-stop:
			g.logger.Info("stop signal received, grpc event dispatching stopped")
			break loop
		case events, ok := <-g.provisioner.EventsChannel():
			if !ok {
				break loop
			}
			g.storeEvents(events)
		}
	}

	return nil
}

func (g *Agent) storeEvents(events []types.Event) {
	if len(events) == 0 {
		return
	}
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)

	g.logger.Debugw("store new events: begin")
	for _, event := range events {
		key := event.Key
		if event.Type == types.EventDelete {
			g.DataStorage.Delete(key)
		} else {
			data, err := json.Marshal(event.Object)
			if err != nil {
				g.logger.Errorw("failed to marshal events",
					zap.Error(err),
				)
				continue
			}
			dataStr := string(data)
			g.DataStorage.Store(key, dataStr)
			g.logger.Debugw("store new events",
				zap.String("key", key),
				zap.String("value", dataStr),
			)
		}
	}
	g.logger.Debugw("store new events: end")

	g.VersionStorage.Store("version", timestamp)

	g.logger.Debugw("store new events: mark version",
		zap.String("version", timestamp),
	)
}
