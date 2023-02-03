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
	"sync"

	"github.com/api7/amesh/controller/pkg/metrics"
)

type ProxyInstance struct {
	UpdateNotifyChan chan struct{}
	//UpdateFunc func() error
}

type InstanceManager struct {
	lock      sync.RWMutex
	instances map[string]*ProxyInstance
	collector metrics.Collector
}

func NewInstanceManager(collector metrics.Collector) *InstanceManager {
	return &InstanceManager{
		instances: map[string]*ProxyInstance{},
		collector: collector,
	}
}

func (m *InstanceManager) get(key string) *ProxyInstance {
	m.lock.RLock()
	i, ok := m.instances[key]
	m.lock.RUnlock()

	if ok {
		return i
	}
	return nil
}

func (m *InstanceManager) add(key string, instance *ProxyInstance) {
	m.lock.Lock()
	m.instances[key] = instance
	m.lock.Unlock()
	m.collector.IncManagedInstances()
}

func (m *InstanceManager) delete(key string) {
	m.lock.Lock()
	delete(m.instances, key)
	m.lock.Unlock()
	m.collector.DecManagedInstances()
}

func (m *InstanceManager) foreach(fn func(*ProxyInstance)) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, instance := range m.instances {
		fn(instance)
	}
}
