// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
package utils

import (
	"sync"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/onsi/ginkgo/v2"
)

type MultiError []error

func (multi MultiError) Error() string {
	msg := ""
	for i, err := range multi {
		msg += err.Error()
		if i != len(multi) {
			msg += "\n"
		}
	}
	return msg
}

type parallelExecutor struct {
	name    string
	started time.Time

	wg sync.WaitGroup

	errorsLock sync.Mutex
	errors     MultiError
}

func NewParallelExecutor(name string) *parallelExecutor {
	return &parallelExecutor{
		name:    name,
		started: time.Now(),
	}
}

func (exec *parallelExecutor) Add(handlers ...func()) {
	exec.wg.Add(1)
	go func() {
		defer ginkgo.GinkgoRecover()
		defer exec.wg.Done()

		for _, handler := range handlers {
			handler()
		}
	}()
}

func (exec *parallelExecutor) AddE(handlers ...func() error) {
	exec.wg.Add(1)
	go func() {
		defer ginkgo.GinkgoRecover()
		defer exec.wg.Done()

		for _, handler := range handlers {
			err := handler()
			if err != nil {
				exec.errorsLock.Lock()
				exec.errors = append(exec.errors, err)
				exec.errorsLock.Unlock()
			}
		}
	}()
}

func (exec *parallelExecutor) Wait() {
	exec.wg.Wait()
	if exec.name != "" {
		log.SkipFramesOnce(1)
		LogTimeTrack(exec.started, exec.name+" end (%v)")
	}
}

func (exec *parallelExecutor) Errors() MultiError {
	return exec.errors
}
