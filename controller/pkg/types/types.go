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

package types

import (
	"k8s.io/apimachinery/pkg/util/sets"

	ameshv1alpha1 "github.com/api7/amesh/apis/amesh/v1alpha1"
)

type PodPluginConfig struct {
	Plugins []ameshv1alpha1.AmeshPluginConfigPlugin
	Version string
}

type PodPluginConfigCache interface {
	GetPodPluginConfigs(key string) ([]*PodPluginConfig, error)
}

type UpdatePodPluginConfigEvent struct {
	Namespace string
	Pods      sets.String
	//Plugins   []ameshv1alpha1.AmeshPluginConfigPlugin
}

type PodChangeNotifier interface {
	AddPodChangeListener(receiver PodChangeReceiver)
}

type PodChangeReceiver interface {
	NotifyPodChange(*UpdatePodPluginConfigEvent)
}
