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

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
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
