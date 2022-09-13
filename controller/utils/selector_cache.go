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

package utils

import (
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	apisixv1alpha1lister "github.com/api7/amesh/api/client/listers/apis/v1alpha1"
)

// SelectorCache is a cache of selectors to avoid high CPU consumption caused by frequent calls
type SelectorCache struct {
	lock sync.RWMutex
	// key -> selector
	cache map[string]labels.Selector
}

// NewSelectorCache init SelectorCache for controller.
func NewSelectorCache() *SelectorCache {
	return &SelectorCache{
		cache: map[string]labels.Selector{},
	}
}

// Get return selector and existence in SelectorCache by key.
func (sc *SelectorCache) Get(key string) (labels.Selector, bool) {
	sc.lock.RLock()
	selector, ok := sc.cache[key]
	sc.lock.RUnlock()
	return selector, ok
}

// Update can update or add a selector in SelectorCache while plugin config's selector changed.
func (sc *SelectorCache) Update(key string, selector *metav1.LabelSelector) error {
	if len(selector.MatchLabels)+len(selector.MatchExpressions) == 0 {
		return nil
	}

	sc.lock.Lock()
	defer sc.lock.Unlock()

	s, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return err
	}
	sc.cache[key] = s

	return nil
}

// Delete can delete selector which exist in SelectorCache.
func (sc *SelectorCache) Delete(key string) {
	sc.lock.Lock()
	delete(sc.cache, key)
	sc.lock.Unlock()
}

func (sc *SelectorCache) GetPodMemberships(lister apisixv1alpha1lister.AmeshPluginConfigLister, pod *v1.Pod) {
	//selector := sc.cache["key"]
	//selector.MatchExpressions[0].
}
