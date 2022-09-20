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

package framework

import (
	"context"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateConfigMap create a ConfigMap object which filled by the key/value
// specified by the caller.
func (f *Framework) CreateConfigMap(name, key, value string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string]string{
			key: value,
		},
	}
	client, err := k8s.GetKubernetesClientFromOptionsE(ginkgo.GinkgoT(), f.kubectlOpts)
	if err != nil {
		return err
	}
	if _, err := client.CoreV1().ConfigMaps(f.namespace).Create(context.TODO(), cm, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

// CreateResourceFromString creates a Kubernetes resource from the given manifest.
func (f *Framework) CreateResourceFromString(res string) error {
	return k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, res)
}

func (f *Framework) WaitForNamespaceDeletion(namespace string) {
	ns, err := k8s.GetNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, namespace)

	if err == nil {
		if ns.DeletionTimestamp != nil {
			// wait for deletion
			log.Infof("namespace %s is deleting, wait", namespace)

			condFunc := func() (bool, error) {
				_, err := k8s.GetNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, namespace)
				if err != nil {
					if apierrors.IsNotFound(err) {
						return true, nil
					} else {
						return false, err
					}
				}
				log.Infof("namespace %s is deleting, waiting...", namespace)
				return false, nil
			}
			err = waitExponentialBackoff(condFunc)
			assert.Nil(ginkgo.GinkgoT(), err, "wait for namespace deletion")
		}
	} else if !apierrors.IsNotFound(err) {
		assert.Nil(ginkgo.GinkgoT(), err, "get namespace")
	}
}
