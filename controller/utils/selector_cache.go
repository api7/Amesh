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
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"

	ameshv1alpha1lister "github.com/api7/amesh/apis/client/listers/amesh/v1alpha1"
)

// SelectorCache is a cache of selectors to avoid high CPU consumption caused by frequent calls
type SelectorCache struct {
	lister ameshv1alpha1lister.AmeshPluginConfigLister

	lock sync.RWMutex
	// key -> selector
	cache map[string]labels.Selector
}

// NewSelectorCache init SelectorCache for controller.
func NewSelectorCache(lister ameshv1alpha1lister.AmeshPluginConfigLister) *SelectorCache {
	return &SelectorCache{
		lister: lister,
		cache:  map[string]labels.Selector{},
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
func (sc *SelectorCache) Update(key string, metaSelector *metav1.LabelSelector) (selector labels.Selector, err error) {
	if metaSelector == nil || (len(metaSelector.MatchLabels)+len(metaSelector.MatchExpressions) == 0) {
		return nil, nil
	} else {
		selector, err = metav1.LabelSelectorAsSelector(metaSelector)
		if err != nil {
			return nil, err
		}
	}

	sc.lock.Lock()
	sc.cache[key] = selector
	sc.lock.Unlock()

	return selector, nil
}

// Delete can delete selector which exist in SelectorCache.
func (sc *SelectorCache) Delete(key string) {
	sc.lock.Lock()
	delete(sc.cache, key)
	sc.lock.Unlock()
}

func (sc *SelectorCache) GetPodPluginConfigs(pod *v1.Pod) (sets.String, error) {
	matchedConfigKeys := sets.String{}

	configs, err := sc.lister.AmeshPluginConfigs(pod.Namespace).List(labels.Everything())
	if err != nil {
		return matchedConfigKeys, err
	}

	for _, config := range configs {
		key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(config)
		if err != nil {
			return nil, err
		}

		if config.Spec.Selector == nil {
			// nil is match every thing so add it directly
			matchedConfigKeys.Insert(key)
			continue
		}

		var selector labels.Selector
		if v, ok := sc.Get(key); ok {
			selector = v
		} else {
			selector, err = sc.Update(key, config.Spec.Selector)
			if err != nil {
				return nil, err
			}
		}

		if selector.Matches(labels.Set(pod.Labels)) {
			matchedConfigKeys.Insert(key)
		}
	}

	return matchedConfigKeys, nil
}
