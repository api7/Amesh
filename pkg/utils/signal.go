// Copyright 2021 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/api7/gopkg/pkg/log"
)

// WaitForSignal creates a channel and wait until desired signal
// arriving. It's a block call so be sure you're using it correctly.
func WaitForSignal(callback func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

	sig := <-sigCh
	log.Infof("signal arrived: %s", sig.String())

	callback()
}
