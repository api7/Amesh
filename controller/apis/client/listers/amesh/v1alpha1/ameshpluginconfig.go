/*
Copyright 2022 The Amesh Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/api7/amesh/apis/amesh/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// AmeshPluginConfigLister helps list AmeshPluginConfigs.
// All objects returned here must be treated as read-only.
type AmeshPluginConfigLister interface {
	// List lists all AmeshPluginConfigs in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.AmeshPluginConfig, err error)
	// AmeshPluginConfigs returns an object that can list and get AmeshPluginConfigs.
	AmeshPluginConfigs(namespace string) AmeshPluginConfigNamespaceLister
	AmeshPluginConfigListerExpansion
}

// ameshPluginConfigLister implements the AmeshPluginConfigLister interface.
type ameshPluginConfigLister struct {
	indexer cache.Indexer
}

// NewAmeshPluginConfigLister returns a new AmeshPluginConfigLister.
func NewAmeshPluginConfigLister(indexer cache.Indexer) AmeshPluginConfigLister {
	return &ameshPluginConfigLister{indexer: indexer}
}

// List lists all AmeshPluginConfigs in the indexer.
func (s *ameshPluginConfigLister) List(selector labels.Selector) (ret []*v1alpha1.AmeshPluginConfig, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AmeshPluginConfig))
	})
	return ret, err
}

// AmeshPluginConfigs returns an object that can list and get AmeshPluginConfigs.
func (s *ameshPluginConfigLister) AmeshPluginConfigs(namespace string) AmeshPluginConfigNamespaceLister {
	return ameshPluginConfigNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// AmeshPluginConfigNamespaceLister helps list and get AmeshPluginConfigs.
// All objects returned here must be treated as read-only.
type AmeshPluginConfigNamespaceLister interface {
	// List lists all AmeshPluginConfigs in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.AmeshPluginConfig, err error)
	// Get retrieves the AmeshPluginConfig from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.AmeshPluginConfig, error)
	AmeshPluginConfigNamespaceListerExpansion
}

// ameshPluginConfigNamespaceLister implements the AmeshPluginConfigNamespaceLister
// interface.
type ameshPluginConfigNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all AmeshPluginConfigs in the indexer for a given namespace.
func (s ameshPluginConfigNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.AmeshPluginConfig, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AmeshPluginConfig))
	})
	return ret, err
}

// Get retrieves the AmeshPluginConfig from the indexer for a given namespace and name.
func (s ameshPluginConfigNamespaceLister) Get(name string) (*v1alpha1.AmeshPluginConfig, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("ameshpluginconfig"), name)
	}
	return obj.(*v1alpha1.AmeshPluginConfig), nil
}
