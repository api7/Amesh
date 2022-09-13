/*
Copyright 2022.

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

package amesh

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	v1informer "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ameshv1alpha1 "github.com/api7/amesh/apis/amesh/v1alpha1"
	ameshv1alpha1informer "github.com/api7/amesh/apis/client/informers/externalversions/amesh/v1alpha1"
	"github.com/api7/amesh/pkg/types"
	"github.com/api7/amesh/utils"
)

var (
	_ types.PodPluginConfigCache = (*AmeshPluginConfigReconciler)(nil)
)

// AmeshPluginConfigReconciler reconciles a AmeshPluginConfig object
type AmeshPluginConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	podInformer      v1informer.PodInformer
	selectorCache    *utils.SelectorCache
	pluginsCacheLock sync.RWMutex
	pluginsCache     map[string][]ameshv1alpha1.AmeshPluginConfigPlugin // TODO: Potential High Memory Usage
}

func NewAmeshPluginConfigController(cli client.Client, scheme *runtime.Scheme,
	podInformer v1informer.PodInformer,
	ameshPluginConfigInformer ameshv1alpha1informer.AmeshPluginConfigInformer) *AmeshPluginConfigReconciler {
	c := &AmeshPluginConfigReconciler{
		Client: cli,
		Log:    ctrl.Log.WithName("controllers").WithName("AmeshPluginConfig"),
		Scheme: scheme,

		podInformer:   podInformer,
		selectorCache: utils.NewSelectorCache(ameshPluginConfigInformer.Lister()),
		pluginsCache:  map[string][]ameshv1alpha1.AmeshPluginConfigPlugin{},
	}
	return c
}

//+kubebuilder:rbac:groups=apisix.apache.org,resources=ameshpluginconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apisix.apache.org,resources=ameshpluginconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apisix.apache.org,resources=ameshpluginconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *AmeshPluginConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	r.Log.V(4).Info("reconciling amesh plugin config", "namespace", req.Namespace, "name", req.Name)

	instance := &ameshv1alpha1.AmeshPluginConfig{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// requeue
		return ctrl.Result{}, errors.Wrap(err, "get object")
	}
	if instance.DeletionTimestamp != nil {
		r.Log.Error(err, "unexpected DeletionTimestamp")
		// don't requeue
		return ctrl.Result{}, nil
	}

	key := req.NamespacedName.String()
	oldSelector, ok := r.selectorCache.Get(key)
	newSelector, err := r.selectorCache.Update(key, instance.Spec.Selector)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.pluginsCacheLock.Lock()
	r.pluginsCache[key] = instance.Spec.Plugins
	r.pluginsCacheLock.Unlock()

	pluginChanged := true

	// TODO: detect if plugins are changed
	// TODO: detect if selectors are changed
	// TODO: detect affected pods
	_ = oldSelector
	if !ok && newSelector == nil {
		// TODO: both empty (everything), if plugins changed, then affects all pods

		return ctrl.Result{}, nil
	} else if !ok {
		if pluginChanged {
			// TODO: update all because plugin changed
		} else {
			// TODO: previous is everything, should remove their config
			// TODO: plugin doesn't changed, so new-matched pods do nothing
		}
	} else if newSelector == nil {
		if pluginChanged {
			// TODO: update all because plugin changed
		} else {
			// TODO: new is everything, should add config for them
			// TODO: plugin doesn't changed, so previous-matched pods do nothing
		}
	} else {
		// TODO: calculate intersection
		// TODO: Remove unmatched Pods
		if pluginChanged {
			// TODO: update matched Pods
		}
	}
	// TODO: If Pod labels changed, Re-send

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AmeshPluginConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ameshv1alpha1.AmeshPluginConfig{}).
		Complete(r)
}

func (r *AmeshPluginConfigReconciler) GetPodPluginConfigs(key string) ([]ameshv1alpha1.AmeshPluginConfigPlugin, error) {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil, err
	}
	pod, err := r.podInformer.Lister().Pods(ns).Get(name)
	if err != nil {
		return nil, err
	}
	pluginConfigs, err := r.selectorCache.GetPodPluginConfigs(pod)
	if err != nil {
		return nil, err
	}

	var plugins []ameshv1alpha1.AmeshPluginConfigPlugin
	r.pluginsCacheLock.RLock()
	for _, key := range pluginConfigs.List() {
		cached, ok := r.pluginsCache[key]
		if !ok {
			continue
		}
		plugins = append(plugins, cached...)
	}
	r.pluginsCacheLock.RUnlock()

	return plugins, nil
}
