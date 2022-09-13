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

package pkg

import (
	"net"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	ctrl "sigs.k8s.io/controller-runtime"

	protov1 "github.com/api7/amesh/api/proto/v1"
	"github.com/api7/amesh/pkg/types"
)

var (
	_ protov1.AmeshServiceServer = (*GRPCController)(nil)
)

type GRPCController struct {
	protov1.UnimplementedAmeshServiceServer
	Log logr.Logger

	stopCh       <-chan struct{}
	grpcListener net.Listener
	grpcSrv      *grpc.Server

	instanceManager   *InstanceManager
	pluginConfigCache types.PodPluginConfigCache
}

func NewGRPCController(GRPCServerAddr string, pluginConfigCache types.PodPluginConfigCache) (*GRPCController, error) {
	c := &GRPCController{
		Log: ctrl.Log.WithName("controllers").WithName("GRPCServer"),

		instanceManager:   NewInstanceManager(),
		pluginConfigCache: pluginConfigCache,
	}

	grpcListener, err := net.Listen("tcp", GRPCServerAddr)
	if err != nil {
		return nil, errors.Wrap(err, "bad grpc server addr")
	}
	c.Log.Info("starting grpc server", "addr", GRPCServerAddr)
	c.grpcListener = grpcListener

	// TODO expose configurations for the server keepalive parameters.
	params := keepalive.ServerParameters{
		MaxConnectionIdle: 15 * time.Minute,
	}

	c.grpcSrv = grpc.NewServer(grpc.KeepaliveParams(params))
	protov1.RegisterAmeshServiceServer(c.grpcSrv, c)

	return c, nil
}

func (c *GRPCController) Run(eventChan <-chan *types.UpdatePodPluginConfigEvent, stopCh <-chan struct{}) {
	c.stopCh = stopCh

	go func() {
		err := c.grpcSrv.Serve(c.grpcListener)
		if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			c.Log.Error(err, "grpc server serve loop aborted")
		}
	}()

	go func() {
		for {
			select {
			case updateEvent := <-eventChan:
				for podName := range updateEvent.Pods {
					// TODO: too many lock/unlock
					instance := c.instanceManager.get(updateEvent.Namespace + "/" + podName)
					if instance != nil {
						go func() {
							instance.UpdateNotifyChan <- struct{}{} // updateEvent.Plugins
						}()
					}
				}
			case <-stopCh:
				c.Log.Info("stop signal received, update loop stopping")
				return
			}
		}
	}()

	c.Log.Info("grpc server is running")
	<-stopCh
	c.Log.Info("stop signal received, grpc server stopping")
	if err := c.grpcListener.Close(); err != nil {
		c.Log.Error(err, "failed to close grpc listener")
	}

	c.Log.Info("grpc server stopped")
}

func (c *GRPCController) sendPodPluginConfig(podKey string, srv protov1.AmeshService_StreamPluginsServer) error {
	configs, err := c.pluginConfigCache.GetPodPluginConfigs(podKey)
	if err != nil {
		c.Log.V(4).Error(err, "failed to query plugin configs", "pod", podKey)
		return status.Errorf(codes.Aborted, err.Error())
	}

	var pluginConfigs []*protov1.PluginConfig
	for _, config := range configs {
		pluginConfig := &protov1.PluginConfig{
			Plugins: []*protov1.Plugin{},
			Version: config.Version,
		}
		for _, plugin := range config.Plugins {
			pluginConfig.Plugins = append(pluginConfig.Plugins, &protov1.Plugin{
				Type:   string(plugin.Type),
				Name:   plugin.Name,
				Config: plugin.Config,
			})
		}

		pluginConfigs = append(pluginConfigs, pluginConfig)
	}

	err = srv.Send(&protov1.PluginsResponse{
		ErrorMessage: nil,
		Plugins:      pluginConfigs,
	})
	if err != nil {
		c.Log.V(4).Error(err, "failed to send PluginsResponse", "pod", podKey)
		return status.Errorf(codes.Aborted, err.Error())
	}

	return nil
}

func (c *GRPCController) StreamPlugins(req *protov1.PluginsRequest, srv protov1.AmeshService_StreamPluginsServer) error {
	podKey := req.GetInstance().GetKey()

	instance := &ProxyInstance{
		UpdateNotifyChan: make(chan struct{}),
		//UpdateFunc: func() error {
		//	return c.sendPodPluginConfig(podKey, srv)
		//},
	}
	c.instanceManager.add(podKey, instance)
	defer c.instanceManager.delete(podKey)

	// initial send
	err := c.sendPodPluginConfig(podKey, srv)
	if err != nil {
		return err
	}

	for {
		select {
		case <-instance.UpdateNotifyChan:
			err := c.sendPodPluginConfig(podKey, srv)
			if err != nil {
				return err
			}

		case <-srv.Context().Done():
			c.Log.Info("AmeshService grpc server context done", "pod", podKey)
			return nil
		case <-c.stopCh:
			c.Log.Info("grpc server stopped, exited", "pod", podKey)
			return nil
		}
	}
}
