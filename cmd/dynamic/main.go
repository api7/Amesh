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
package main

import "C"

import (
	"context"
	"os"
	"strconv"
	"unsafe"

	"github.com/api7/gopkg/pkg/log"

	"github.com/api7/amesh/pkg/amesh"
	"github.com/api7/amesh/pkg/utils"
)

func main() {
}

//export Log
func Log(msg string) {
	log.Infof(msg)
}

//export StartTestAmesh
func StartTestAmesh(src string) {
	ctx, cancel := context.WithCancel(context.Background())
	agent, err := amesh.NewAgent(ctx, src, "", 0, nil, nil, "debug", "stderr")
	if err != nil {
		utils.Dief("failed to create generator: %v", err.Error())
	}

	_ = cancel
	go func() {
		utils.WaitForSignal(func() {
			cancel()
		})
	}()

	go func() {
		if err = agent.Run(ctx.Done()); err != nil {
			utils.Dief("agent error: %v", err.Error())
		}
	}()
}

//export initial
func initial(dataZone, versionZone unsafe.Pointer) {
	src := os.Getenv("ISTIO_XDS_SOURCE")
	if src == "" {
		src = "grpc://istiod.istio-system.svc.cluster.local:15010"
	}
	ameshGrpc := os.Getenv("AMESH_GRPC_SOURCE")
	if ameshGrpc == "" {
		ameshGrpc = "grpc://amesh-controller.istio-system.svc.cluster.local:15810"
	}
	syncInterval := os.Getenv("AMESH_SYNC_INTERVAL")
	if syncInterval == "" {
		syncInterval = "10"
	}
	interval, err := strconv.Atoi(syncInterval)
	if err != nil {
		utils.Dief("parse interval failed: %v", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	agent, err := amesh.NewAgent(ctx, src, ameshGrpc, interval, dataZone, versionZone, "debug", "stderr")
	if err != nil {
		utils.Dief("failed to create generator: %v", err.Error())
	}

	_ = cancel
	go func() {
		utils.WaitForSignal(func() {
			cancel()
		})
	}()

	go func() {
		if err = agent.Run(ctx.Done()); err != nil {
			utils.Dief("agent error: %v", err.Error())
		}
	}()
}
