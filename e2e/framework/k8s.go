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
	"fmt"
	"strings"

	"github.com/api7/gopkg/pkg/log"
	"github.com/fatih/color"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"go.uber.org/zap"
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

// ApplyResourceFromString creates a Kubernetes resource from the given manifest.
func (f *Framework) ApplyResourceFromString(res string) error {
	return k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, res)
}

func (f *Framework) GetDeploymentPodNames(namespace, name string) ([]string, error) {
	return f.GetPodNamesByLabel(namespace, "app="+name)
}

func (f *Framework) GetPodNamesByLabel(namespace string, labelSelector string) ([]string, error) {
	var names []string

	client, err := k8s.GetKubernetesClientFromOptionsE(ginkgo.GinkgoT(), f.kubectlOpts)
	if err != nil {
		return names, err
	}
	pods, err := client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return names, err
	}

	for _, pod := range pods.Items {
		names = append(names, pod.Name)
	}
	return names, nil
}

// GetServiceFQDN returns the FQDN description for HttpBin service.
func (f *Framework) GetServiceFQDN(name string) string {
	return fmt.Sprintf("%s.%s.svc", name, f.namespace)
}

// DeleteResourceFromString deletes a Kubernetes resource from the given manifest.
func (f *Framework) DeleteResourceFromString(res, name string) error {
	_, err := f.RunKubectlCommand("delete", res, name)
	return err
}

func (f *Framework) DeleteResource(resourceType, namespace string, args ...string) error {
	cmd := []string{"-n", namespace, "delete", resourceType}
	cmd = append(cmd, args...)
	err := k8s.RunKubectlE(f.t, f.kubectlOpts, cmd...)

	log.SkipFrames(1)
	defer log.SkipFrames(-1)
	log.Info("Executing command: kubectl " + strings.Join(cmd, " "))
	if err != nil {
		log.Errorw("delete resource failed",
			zap.Error(err),
			zap.String("namespace", namespace),
			zap.String("resource", resourceType),
			zap.Strings("args", args),
		)
	}
	return err
}

// DeletePod deletes a Kubernetes pod from the given namespace and name.
func (f *Framework) DeletePod(namespace, name string) error {
	log.SkipFrames(1)
	defer log.SkipFrames(-1)
	err := f.DeleteResource("pod", namespace, name)
	return err
}

// DeletePod deletes a Kubernetes pod from the given namespace and labels.
func (f *Framework) DeletePodByLabel(namespace string, labels ...string) error {
	var labelArgs []string
	for _, label := range labels {
		labelArgs = append(labelArgs, "-l", label)
	}
	log.SkipFrames(1)
	defer log.SkipFrames(-1)
	err := f.DeleteResource("pod", namespace, labelArgs...)
	return err
}

func (f *Framework) WaitForServiceReady(ns, name string) (string, error) {
	log.SkipFrames(1)
	defer log.SkipFrames(-1)
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

func (f *Framework) WaitForDeploymentPodsReady(name string, namespace ...string) error {
	ns := f.kubectlOpts.Namespace
	if len(namespace) > 0 {
		ns = namespace[0]
	}

	log.SkipFrames(1)
	defer log.SkipFrames(-1)
	return utils.WaitForDeploymentPodsReady(f.kubectlOpts, ns, name)
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

func (f *Framework) GetDeploymentLogs(ns, name string) string {
	cli, err := k8s.GetKubernetesClientFromOptionsE(f.t, f.kubectlOpts)
	if err != nil {
		utils.AssertNil(err, "get client error: %s", err.Error())
		return ""
	}

	pods, err := cli.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=" + name,
	})
	if err != nil {
		return ""
	}

	var buf strings.Builder
	buf.WriteString(color.RedString("=== Dumping Deployment ===\n"))
	for _, pod := range pods.Items {
		buf.WriteString(color.GreenString("=== Pod: %s ===\n", pod.Name))
		for _, container := range pod.Spec.Containers {
			buf.WriteString(color.CyanString("=== Container: %s ===\n", container.Name))
			logs, err := cli.CoreV1().RESTClient().Get().
				Resource("pods").
				Namespace(ns).
				Name(pod.Name).SubResource("log").
				Param("container", container.Name).
				Do(context.TODO()).
				Raw()
			if err != nil {
				buf.WriteString(color.RedString("Error: failed to retrieve logs: %s", err.Error()))
			} else {
				buf.Write(logs)
			}
		}
		buf.WriteString(color.GreenString("\n=== Pod End ===\n"))
	}
	buf.WriteString(color.RedString("\n=== Deployment End ===\n"))
	return buf.String()
}

func (f *Framework) RunKubectlCommand(args ...string) (string, error) {
	log.Infof("Running command: kubectl -n %v %v", f.kubectlOpts.Namespace, strings.Join(args, " "))
	return k8s.RunKubectlAndGetOutputE(ginkgo.GinkgoT(), f.kubectlOpts, args...)
}
