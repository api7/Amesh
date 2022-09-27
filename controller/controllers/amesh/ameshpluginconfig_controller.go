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
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	v1informer "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ameshv1alpha1 "github.com/api7/amesh/controller/apis/amesh/v1alpha1"
	clientset "github.com/api7/amesh/controller/apis/client/clientset/versioned"
	ameshv1alpha1informer "github.com/api7/amesh/controller/apis/client/informers/externalversions/amesh/v1alpha1"
	"github.com/api7/amesh/controller/pkg/types"
	"github.com/api7/amesh/controller/utils"
)

var (
	_ types.PodPluginConfigCache = (*AmeshPluginConfigReconciler)(nil)
)

// AmeshPluginConfigReconciler reconciles a AmeshPluginConfig object
type AmeshPluginConfigReconciler struct {
	client.Client
	clientset clientset.Interface
	recorder  record.EventRecorder

	Log    logr.Logger
	Scheme *runtime.Scheme

	podInformer      v1informer.PodInformer
	selectorCache    *utils.SelectorCache
	pluginsCacheLock sync.RWMutex

	// PluginConfig key -> config
	pluginsCache map[string]*types.PodPluginConfig // TODO: Potential High Memory Usage

	subsLock sync.RWMutex
	subs     []types.PodChangeReceiver
}

func NewAmeshPluginConfigController(cli client.Client, scheme *runtime.Scheme,
	clientset clientset.Interface, kubeClient kubernetes.Interface,
	podInformer v1informer.PodInformer,
	ameshPluginConfigInformer ameshv1alpha1informer.AmeshPluginConfigInformer) *AmeshPluginConfigReconciler {

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})

	c := &AmeshPluginConfigReconciler{
		Client:    cli,
		clientset: clientset,
		recorder:  eventBroadcaster.NewRecorder(scheme, v1.EventSource{Component: "AmeshPluginConfigReconciler"}),

		Log:    ctrl.Log.WithName("controllers").WithName("AmeshPluginConfig"),
		Scheme: scheme,

		podInformer:   podInformer,
		selectorCache: utils.NewSelectorCache(ameshPluginConfigInformer.Lister()),
		pluginsCache:  map[string]*types.PodPluginConfig{},
	}

	// TODO: FIXME: delay after AmeshPluginController ready
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod, ok := obj.(*v1.Pod)
			if ok {
				c.Log.Info("Pod added, notify", "pod", pod.Name)
				c.SendPluginsConfigs(pod.Namespace, sets.NewString(pod.Name), nil)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			old, ok1 := oldObj.(*v1.Pod)
			pod, ok2 := newObj.(*v1.Pod)
			if ok1 && ok2 {
				if pod.ResourceVersion > old.ResourceVersion && !utils.LabelsEqual(old.Labels, pod.Labels) {
					c.Log.Info("Pod label changed, notify", "pod", pod.Name)
					// todo shall we check if it really changes?
					c.SendPluginsConfigs(pod.Namespace, sets.NewString(pod.Name), nil)
				}
			}
		},
	})

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

	key := req.NamespacedName.String()
	r.Log.Info("reconciling amesh plugin config", "namespace", req.Namespace, "name", req.Name)

	instance := &ameshv1alpha1.AmeshPluginConfig{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue

			r.Log.Error(err, "config plugin deleted", "key", key)
			r.pluginsCacheLock.Lock()
			delete(r.pluginsCache, key)
			r.pluginsCacheLock.Unlock()

			oldSelector, ok := r.selectorCache.Get(key)
			if ok {
				pods, err := r.podInformer.Lister().Pods(req.Namespace).List(oldSelector)
				if err != nil {
					return ctrl.Result{}, err
				}
				podSet := sets.NewString()
				for _, pod := range pods {
					podSet.Insert(pod.Name)
				}
				r.SendPluginsConfigs(req.Namespace, podSet, nil)
			}

			r.selectorCache.Delete(key)

			return ctrl.Result{}, nil
		}
		// requeue
		return ctrl.Result{}, err
	}
	if instance.DeletionTimestamp != nil {
		r.Log.Info("DeletionTimestamp found, skipped", "key", req.NamespacedName)
		// don't requeue
		return ctrl.Result{}, nil
	}

	r.pluginsCacheLock.RLock()
	oldConfig, ok2 := r.pluginsCache[key]
	r.pluginsCacheLock.RUnlock()
	if ok2 && oldConfig.Version >= instance.ResourceVersion {
		return ctrl.Result{}, nil
	}

	oldSelector, hasOldSelector := r.selectorCache.Get(key)
	newSelector, err := r.selectorCache.Update(key, instance.Spec.Selector)
	if err != nil {
		return ctrl.Result{}, err
	}

	newPlugins := instance.Spec.Plugins
	newConfig := &types.PodPluginConfig{
		Name:    key,
		Plugins: newPlugins,
		Version: instance.ResourceVersion,
	}
	r.pluginsCacheLock.Lock()
	r.pluginsCache[key] = newConfig
	r.pluginsCacheLock.Unlock()

	r.Log.Info("plugin updated", "key", key, "config", newConfig)

	if !hasOldSelector {
		oldSelector = labels.Everything()
	}
	onlyInOld, both, onlyInNew, err := utils.DiffPods(r.podInformer.Lister(), req.Namespace, oldSelector, newSelector)
	if err != nil {
		return ctrl.Result{}, err
	}

	pluginChanged := true // Defaults true to ensure no data missed
	if ok2 {
		pluginChanged = !utils.PluginsConfigEqual(oldConfig.Plugins, newPlugins)
	}

	if !hasOldSelector && newSelector == nil {
		// both empty (everything), if plugins changed, then affects all pods; else do nothing
		if pluginChanged {
			// affect all Pods
			allPods, err := r.podInformer.Lister().Pods(req.Namespace).List(labels.Everything())
			if err != nil {
				return ctrl.Result{}, err
			}
			pods := sets.String{}
			for _, pod := range allPods {
				pods.Insert(pod.Name)
			}
			r.SendPluginsConfigs(req.Namespace, pods, newPlugins)
		}
	} else if !hasOldSelector {
		// previous is everything, should remove config for no longer matched
		r.SendPluginsConfigs(req.Namespace, onlyInOld, nil)
		if pluginChanged {
			// update still matched
			r.SendPluginsConfigs(req.Namespace, both, newPlugins)
			r.SendPluginsConfigs(req.Namespace, onlyInNew, newPlugins)
		}
	} else if newSelector == nil {
		// current is everything, should add config for newly matched
		r.SendPluginsConfigs(req.Namespace, onlyInNew, newPlugins)
		if pluginChanged {
			// update still matched
			r.SendPluginsConfigs(req.Namespace, onlyInOld, newPlugins)
			r.SendPluginsConfigs(req.Namespace, both, newPlugins)
		}
	} else {
		// Remove no longer matched Pods
		r.SendPluginsConfigs(req.Namespace, onlyInOld, nil)
		// Add newly matched Pods
		r.SendPluginsConfigs(req.Namespace, onlyInNew, newPlugins)
		if pluginChanged {
			// Update existed Pods
			r.SendPluginsConfigs(req.Namespace, both, newPlugins)
		}
	}

	r.recordStatus(instance, utils.ResourceReconciled, nil, metav1.ConditionTrue)

	return ctrl.Result{}, nil
}

func (r *AmeshPluginConfigReconciler) AddPodChangeListener(receiver types.PodChangeReceiver) {
	r.subsLock.Lock()
	r.subs = append(r.subs, receiver)
	r.subsLock.Unlock()
}

// SendPluginsConfigs triggers a re-sync process of the pods.
// Currently, we don't count the plugins passed, actual configs are retrieved from GetPodPluginConfigs
func (r *AmeshPluginConfigReconciler) SendPluginsConfigs(ns string, names sets.String, plugins []ameshv1alpha1.AmeshPluginConfigPlugin) {
	if names.Len() <= 0 {
		return
	}
	r.Log.Info("notify plugins config changed", "ns", ns, "pods", names.List())

	r.subsLock.RLock()
	defer r.subsLock.RUnlock()

	for _, sub := range r.subs {
		sub.NotifyPodChange(&types.UpdatePodPluginConfigEvent{
			Namespace: ns,
			Pods:      names,
			//Plugins:   plugins,
		})
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *AmeshPluginConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ameshv1alpha1.AmeshPluginConfig{}).
		Complete(r)
}

func (r *AmeshPluginConfigReconciler) GetPodPluginConfigs(key string) ([]*types.PodPluginConfig, error) {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
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

	var configs []*types.PodPluginConfig
	r.pluginsCacheLock.RLock()
	for _, key := range pluginConfigs.List() {
		cached, ok := r.pluginsCache[key]
		if !ok {
			continue
		}
		configs = append(configs, cached)
	}
	r.pluginsCacheLock.RUnlock()

	return configs, nil
}

// recordStatus record resources status
func (r *AmeshPluginConfigReconciler) recordStatus(config *ameshv1alpha1.AmeshPluginConfig, reason string, err error, status metav1.ConditionStatus) {
	// build condition
	message := utils.ConditionSyncSuccess
	if err != nil {
		message = err.Error()
	}
	condition := metav1.Condition{
		Type:               utils.ConditionSync,
		Reason:             reason,
		Status:             status,
		Message:            message,
		ObservedGeneration: config.GetGeneration(),
	}

	config = config.DeepCopy()

	if config.Status.Conditions == nil {
		conditions := make([]metav1.Condition, 0)
		config.Status.Conditions = conditions
	}
	if utils.VerifyGeneration(&config.Status.Conditions, condition) {
		meta.SetStatusCondition(&config.Status.Conditions, condition)
		if _, errRecord := r.clientset.ApisixV1alpha1().AmeshPluginConfigs(config.Namespace).
			UpdateStatus(context.TODO(), config, metav1.UpdateOptions{}); errRecord != nil {
			r.Log.Error(errRecord, "failed to record status change for ApisixClusterConfig", "name", config.Name, "namespace", config.Namespace)
		}
	}
}
