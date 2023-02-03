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

package main

import (
	"flag"
	"github.com/api7/amesh/controller/pkg/metrics"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	ameshv1alpha1 "github.com/api7/amesh/controller/apis/amesh/v1alpha1"
	clientset "github.com/api7/amesh/controller/apis/client/clientset/versioned"
	ameshv1alpha1informers "github.com/api7/amesh/controller/apis/client/informers/externalversions"
	"github.com/api7/amesh/controller/controllers/amesh"
	"github.com/api7/amesh/controller/pkg"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(ameshv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":18080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":18081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Logger:                 ctrl.Log.WithName("Manager"),
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "c979a3f2.apisix.apache.org",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		setupLog.Error(err, "failed to build kubeconfig")
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "failed to create kubernetes clientset")
	}

	ameshClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "failed to create Amesh clientset: %s")
	}
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	ameshInformerFactory := ameshv1alpha1informers.NewSharedInformerFactory(ameshClient, time.Second*30)

	// Controllers

	ameshPluginConfigController := amesh.NewAmeshPluginConfigController(mgr.GetClient(), mgr.GetScheme(),
		ameshClient, kubeClient,
		kubeInformerFactory.Core().V1().Pods(),
		ameshInformerFactory.Apisix().V1alpha1().AmeshPluginConfigs(),
	)
	if err = ameshPluginConfigController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AmeshPluginConfig")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Metrics
	collector := metrics.NewPrometheusCollector()

	// GRPC Server
	grpc, err := pkg.NewGRPCController(":15810", ameshPluginConfigController, collector)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GRPC")
		os.Exit(1)
	}
	ameshPluginConfigController.AddPodChangeListener(grpc)

	// Startup
	ctx := ctrl.SetupSignalHandler()
	kubeInformerFactory.Start(ctx.Done())
	ameshInformerFactory.Start(ctx.Done())
	go grpc.Run(ctx.Done())
	go collector.Run(ctx.Done())

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
