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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/api7/amesh/e2e/framework/utils"
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

// CreateResourceFromString creates a Kubernetes resource from the given manifest.
func (f *Framework) DeleteResourceFromString(res, name string) error {
	_, err := k8s.RunKubectlAndGetOutputE(ginkgo.GinkgoT(), f.kubectlOpts, "delete", res, name)
	return err
}

func (f *Framework) WaitForServiceReady(ns, name string) (string, error) {
	return utils.WaitForServiceReady(f.kubectlOpts, ns, name)
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
				log.Debugf("namespace %s is deleting, waiting...", namespace)
				return false, nil
			}
			err = utils.WaitExponentialBackoff(condFunc)
			utils.AssertNil(err, "wait for namespace deletion")
		}
	} else if !apierrors.IsNotFound(err) {
		utils.AssertNil(err, "get namespace")
	}
}

func (f *Framework) WaitForPodsReady(name string) error {
	opts := metav1.ListOptions{
		LabelSelector: "app=" + name,
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(ginkgo.GinkgoT(), f.kubectlOpts, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			log.Debugf("no %s pods created", name)
			return false, nil
		}
		for _, pod := range items {
			found := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type != corev1.PodReady {
					continue
				}
				found = true
				if cond.Status != corev1.ConditionTrue {
					return false, nil
				}
			}
			if !found {
				return false, nil
			}
		}
		return true, nil
	}
	return utils.WaitExponentialBackoff(condFunc)
}

const (
	failureToleration = 10
)

func (f *Framework) WaitForDeploymentPodsReady(name string) error {
	opts := metav1.ListOptions{
		LabelSelector: "app=" + name,
	}

	deploymentFailures := 0
	podFailures := 0
	condFunc := func() (bool, error) {
		if (deploymentFailures + podFailures) >= 2*failureToleration {
			log.Warnf("waiting %s pods... (%v times)", name, deploymentFailures+podFailures)
		} else {
			log.Debugf("waiting %s pods...", name)
		}
		items, err := k8s.ListPodsE(ginkgo.GinkgoT(), f.kubectlOpts, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			if deploymentFailures >= failureToleration {
				log.Warnf("no %s pods created (%v times)", name, deploymentFailures)
			} else {
				log.Debugf("no %s pods created", name)
			}
			deploymentFailures++
			clientset, err := k8s.GetKubernetesClientFromOptionsE(ginkgo.GinkgoT(), f.kubectlOpts)
			if err != nil {
				return false, err
			}

			deployments, err := clientset.AppsV1().Deployments(f.kubectlOpts.Namespace).List(context.Background(), opts)
			if err != nil {
				return false, err
			}
			if len(deployments.Items) == 0 {
				log.Debugf("no %s deployment created", name)
				return false, nil
			}
			for _, deployment := range deployments.Items {
				for _, cond := range deployment.Status.Conditions {
					if deploymentFailures >= failureToleration {
						log.Warnf("%v: %v", deployment.Name, cond.Message)
					} else {
						log.Debugf("%v: %v", deployment.Name, cond.Message)
					}
				}
			}
			return false, nil
		}
		defer func() { podFailures++ }()
		for _, pod := range items {
			found := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type != corev1.PodReady {
					if podFailures >= failureToleration {
						log.Warnf("pod %s type %s status %s: %s", pod.Name, cond.Type, cond.Status, cond.Message)
					} else {
						log.Debugf("pod %s cond %s", pod.Name, cond.Type)
					}
					continue
				}
				found = true
				if cond.Status != corev1.ConditionTrue {
					if podFailures >= failureToleration {
						log.Warnf("pod %s type %s status %s: %s", pod.Name, cond.Type, cond.Status, cond.Message)
					} else {
						log.Debugf("pod %s status %s", pod.Name, cond.Status)
					}
					return false, nil
				}
			}
			if !found {
				return false, nil
			}
		}
		return true, nil
	}
	return utils.WaitExponentialBackoff(condFunc)
}

func (f *Framework) WaitForAmeshPluginConfigEvents(name string, typ string, status metav1.ConditionStatus) error {
	condFunc := func() (bool, error) {
		item, err := f.AmeshClient.ApisixV1alpha1().AmeshPluginConfigs(f.kubectlOpts.Namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, condition := range item.Status.Conditions {
			if condition.Type == typ && condition.Status == status {
				return true, nil
			}
		}
		return false, nil
	}
	return utils.WaitExponentialBackoff(condFunc)
}
