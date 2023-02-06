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
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/api7/gopkg/pkg/log"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

func NewAgent(ctx context.Context, src, ameshGrpc string, syncInterval int, dataZone, versionZone unsafe.Pointer, logLevel, logOutput string) (*Agent, error) {
	color.NoColor = false

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
		SyncInternal:      syncInterval,
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

func (g *Agent) internalDataHandler(dataType string) func(writer http.ResponseWriter, request *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		data, err := g.provisioner.GetData(dataType)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			_, writeErr := writer.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
			if writeErr != nil {
				g.logger.Fatalw(fmt.Sprintf("failed to write %v error", dataType),
					zap.Error(writeErr),
					zap.NamedError("status_error", err),
				)
			}
			return
		}
		writer.WriteHeader(http.StatusOK)
		_, err = writer.Write([]byte(data))
		if err != nil {
			g.logger.Fatalw("failed to write "+dataType,
				zap.Error(err),
			)
		}
		return
	}
}

func applyHeaders(dest http.Header, src http.Header, keys ...string) {
	for _, key := range keys {
		val := src.Get(key)
		if val != "" {
			dest.Set(key, val)
		}
	}
}

func (g *Agent) getMetrics(url string, header http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	applyHeaders(req.Header, header,
		"Accept",
		"Accept-Encoding",
		"User-Agent",
		"X-Prometheus-Scrape-Timeout-Seconds",
	)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get apisix metrics %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get apisix metrics %s, status code: %v", url, resp.StatusCode)
	}
	metrics, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read apisix metrics body %s: %v", url, err)
	}

	return metrics, nil
}

// isGzipEncoding returns the client requires gzip-encoded content
func isGzipEncoding(header http.Header) bool {
	enc := header.Get("Accept-Encoding")
	encodings := strings.Split(enc, ",")
	for _, encoding := range encodings {
		encoding = strings.TrimSpace(encoding)
		if strings.HasPrefix(encoding, "gzip") {
			return true
		}
	}
	return false
}

var gzipPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(nil)
	},
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

	var statusServer *http.Server
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

		// TODO: Add test to verify status server still work when unable to connect to xds source/amesh source
		mux := http.NewServeMux()
		mux.HandleFunc("/status", g.internalDataHandler("status"))
		mux.HandleFunc("/routes", g.internalDataHandler("routes"))
		mux.HandleFunc("/upstreams", g.internalDataHandler("upstreams"))
		mux.HandleFunc("/plugins", g.internalDataHandler("plugins"))
		mux.HandleFunc("/sds/", func(writer http.ResponseWriter, request *http.Request) {
			name := strings.TrimPrefix(request.URL.Path, "/sds/")
			if name != "" {
				g.provisioner.SendSds(name)
			}

			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte("Sending SDS: " + name))
		})
		statusServer = &http.Server{Addr: fmt.Sprintf(":%v", port), Handler: mux}

		err = statusServer.ListenAndServe()
		if err != nil {
			g.logger.Fatalw("failed to run status server",
				zap.Error(err),
			)
			return
		}
	}()

	var metricsServer *http.Server
	go func() {
		portStr := os.Getenv("METRICS_SERVER_PORT")
		if portStr == "NONE" {
			// Do not start metrics server
			g.logger.Infof("METRICS_SERVER_PORT is NONE, status server won't start")
			return
		} else if portStr == "" {
			portStr = "15090"
		}

		port := 15090
		parsed, err := strconv.ParseInt(portStr, 10, 32)
		if err != nil {
			g.logger.Fatalw("failed to parse status server port, using default",
				zap.Error(err),
				zap.String("value", portStr),
			)
		} else {
			port = int(parsed)
		}

		metricsPath := os.Getenv("METRICS_SERVER_PATH")
		if metricsPath == "" {
			metricsPath = "/stats/prometheus"
		}

		apisixMetricsURL := os.Getenv("APISIX_METRICS_URL")
		if apisixMetricsURL == "" {
			apisixMetricsURL = "http://localhost:9091/stats/prometheus"
		}

		// TODO: Add test to verify status server still work when unable to connect to xds source/amesh source
		mux := http.NewServeMux()
		mux.HandleFunc(metricsPath, func(writer http.ResponseWriter, request *http.Request) {
			promhttp.Handler().ServeHTTP(writer, request)

			apisixMetrics, err := g.getMetrics(apisixMetricsURL, request.Header)
			if err != nil {
				g.logger.Errorw("failed to get apisix metrics",
					zap.Error(err),
				)
				return
			}

			w := io.Writer(writer)
			if isGzipEncoding(request.Header) {
				gz := gzipPool.Get().(*gzip.Writer)
				defer gzipPool.Put(gz)

				gz.Reset(w)
				defer gz.Close()

				w = gz
			}

			if _, err := w.Write(apisixMetrics); err != nil {
				g.logger.Errorw("failed to write apisix metrics",
					zap.Error(err),
				)
			}
		})
		metricsServer = &http.Server{Addr: fmt.Sprintf(":%v", port), Handler: mux}

		g.logger.Infof("metrics server started at :%v, apisix metrics at %v", portStr+metricsPath, apisixMetricsURL)

		err = metricsServer.ListenAndServe()
		if err != nil {
			g.logger.Fatalw("failed to run metrics server",
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
			if statusServer != nil {
				if err := statusServer.Shutdown(context.Background()); err != nil {
					g.logger.Fatalw("failed to shut down status server",
						zap.Error(err),
					)
				}
			}
			if metricsServer != nil {
				if err := metricsServer.Shutdown(context.Background()); err != nil {
					g.logger.Fatalw("failed to shut down metrics server",
						zap.Error(err),
					)
				}
			}
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

	g.logger.Debugw(color.BlueString("store new events: begin"))
	for _, event := range events {
		key := event.Key
		if event.Type == types.EventDelete {
			g.DataStorage.Delete(key)
			g.logger.Debugw("delete key",
				zap.String("key", key),
			)
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
	g.logger.Debugw(color.BlueString("store new events: end"))

	g.VersionStorage.Store("version", timestamp)

	g.logger.Debugw(color.BlueString("store new events: mark version"),
		zap.String("version", timestamp),
	)
}
