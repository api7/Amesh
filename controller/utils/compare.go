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
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	v1lister "k8s.io/client-go/listers/core/v1"

	ameshv1alpha1 "github.com/api7/amesh/controller/apis/amesh/v1alpha1"
)

func PluginsConfigEqual(oldPlugins, newPlugins []ameshv1alpha1.AmeshPluginConfigPlugin) bool {
	if len(oldPlugins) != len(newPlugins) {
		return false
	} else {
		oldPluginMap := map[string]string{}
		for _, plugin := range oldPlugins {
			oldPluginMap[string(plugin.Type)+plugin.Name] = plugin.Config
		}
		newPluginMap := map[string]string{}
		for _, plugin := range newPlugins {
			newPluginMap[string(plugin.Type)+plugin.Name] = plugin.Config
		}

		for key, newConfig := range newPluginMap {
			oldConfig, ok := oldPluginMap[key]
			if !ok {
				return false
			}
			if newConfig != oldConfig {
				return false
			}
		}
	}

	return true
}

func LabelSelectorEqual(oldSelector, newSelector labels.Selector) bool {
	oldReqs, oldSelectable := oldSelector.Requirements()
	newReqs, newSelectable := newSelector.Requirements()
	if oldSelectable != newSelectable {
		return false
	}
	if len(oldReqs) == 0 && len(newReqs) == 0 {
		return true
	}
	if len(oldReqs) != len(newReqs) {
		return false
	} else {
		oldReqMap := map[string]sets.String{}
		for _, req := range oldReqs {
			oldReqMap[req.Key()+string(req.Operator())] = req.Values()
		}
		newReqMap := map[string]sets.String{}
		for _, req := range newReqs {
			newReqMap[req.Key()+string(req.Operator())] = req.Values()
		}

		for key, newValues := range newReqMap {
			oldValues, ok := oldReqMap[key]
			if !ok {
				return false
			}
			if !newValues.Equal(oldValues) {
				return false
			}
			fmt.Printf("%v %v equals %v\n", key, newValues, oldValues)
		}
	}

	return true
}

func LabelsEqual(oldLabels, newLabels map[string]string) bool {
	if len(oldLabels) != len(newLabels) {
		return false
	} else {
		for key, newValues := range newLabels {
			oldValues, ok := oldLabels[key]
			if !ok {
				return false
			}
			if newValues != oldValues {
				return false
			}
		}
	}

	return true
}

func DiffPods(lister v1lister.PodLister, ns string, oldSelector, newSelector labels.Selector) (onlyInOld, both, onlyInNew sets.String, err error) {
	onlyInOld, both, onlyInNew = sets.NewString(), sets.NewString(), sets.NewString()

	oldPods, err := lister.Pods(ns).List(oldSelector)
	if err != nil {
		return
	}

	newPods, err := lister.Pods(ns).List(newSelector)
	if err != nil {
		return
	}

	oldMap := map[string]struct{}{}
	for _, pod := range oldPods {
		oldMap[pod.Name] = struct{}{}
	}
	newMap := map[string]struct{}{}
	for _, pod := range newPods {
		newMap[pod.Name] = struct{}{}
	}

	for _, pod := range oldPods {
		_, inOld := oldMap[pod.Name]
		_, inNew := oldMap[pod.Name]
		if inOld && inNew {
			both.Insert(pod.Name)
		} else {
			onlyInOld.Insert(pod.Name)
		}
	}
	for _, pod := range newPods {
		_, inOld := newMap[pod.Name]
		_, inNew := newMap[pod.Name]
		if inOld && inNew {
			both.Insert(pod.Name)
		} else {
			onlyInNew.Insert(pod.Name)
		}
	}

	return
}
