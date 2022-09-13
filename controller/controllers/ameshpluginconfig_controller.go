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

package controllers

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ameshv1alpha1 "github.com/api7/amesh/api/apis/v1alpha1"
	"github.com/api7/amesh/utils"
)

// AmeshPluginConfigReconciler reconciles a AmeshPluginConfig object
type AmeshPluginConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	selectorCache *utils.SelectorCache

	pluginsCacheLock sync.RWMutex
	pluginsCache     map[string][]ameshv1alpha1.AmeshPluginConfigPlugin
}

func NewAmeshPluginConfigController(cli client.Client, scheme *runtime.Scheme) *AmeshPluginConfigReconciler {
	return &AmeshPluginConfigReconciler{
		Client: cli,
		Log:    ctrl.Log.WithName("controllers").WithName("AmeshPluginConfig"),
		Scheme: scheme,

		selectorCache: utils.NewSelectorCache(),
		pluginsCache:  map[string][]ameshv1alpha1.AmeshPluginConfigPlugin{},
	}
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

	// Fetch the AppService instance
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
		return ctrl.Result{}, nil
	}

	r.pluginsCacheLock.Lock()
	r.pluginsCache[req.NamespacedName.String()] = instance.Spec.Plugins
	r.pluginsCacheLock.Unlock()

	// Send to gRPC client

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AmeshPluginConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ameshv1alpha1.AmeshPluginConfig{}).
		Complete(r)
}
