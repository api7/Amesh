package metrics

import (
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	_namespace = "amesh"
)

var (
	// TODO: make them global?
	instances        *prometheus.GaugeVec
	controllerEvents *prometheus.CounterVec
	controllerErrors *prometheus.CounterVec
)

func init() {
	podName := os.Getenv("POD_NAME")
	podNamespace := os.Getenv("POD_NAMESPACE")
	if podNamespace == "" {
		podNamespace = "default"
	}
	constLabels := prometheus.Labels{
		"controller_pod":       podName,
		"controller_namespace": podNamespace,
	}

	instances = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   _namespace,
			Name:        "managed_instances",
			Help:        "Number of managed instances",
			ConstLabels: constLabels,
		},
		[]string{"resource"},
	)
	controllerEvents = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   _namespace,
			Name:        "events_total",
			Help:        "Number of events handled by the controller",
			ConstLabels: constLabels,
		},
		[]string{"operation", "resource"},
	)
	controllerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   _namespace,
			Name:        "errors_total",
			Help:        "Number of errors handled by the controller",
			ConstLabels: constLabels,
		},
		[]string{"operation", "resource"},
	)

	metrics.Registry.MustRegister(
		instances,
		controllerEvents,
		controllerErrors,
	)
}

type Collector interface {
	Run(stop <-chan struct{})
	IncManagedInstances()
	DecManagedInstances()
	AddEvent(resource, operation string)
}

type collector struct {
	logger logr.Logger

	instances        *prometheus.GaugeVec
	controllerEvents *prometheus.CounterVec
	controllerErrors *prometheus.CounterVec
}

// NewPrometheusCollector creates the Prometheus metrics collector.
// It also registers all internal metric collector to prometheus,
// so do not call this function multiple times.
func NewPrometheusCollector() Collector {
	collector := &collector{
		logger: ctrl.Log.WithName("metrics").WithName("Prometheus"),

		instances:        instances,
		controllerEvents: controllerEvents,
		controllerErrors: controllerErrors,
	}

	return collector
}

func (c *collector) Run(stop <-chan struct{}) {
	// Controller runtime will start the server, do nothing here
}

func (c *collector) IncManagedInstances() {
	c.instances.With(prometheus.Labels{
		"resource": "managed_instances",
	}).Inc()
}

func (c *collector) DecManagedInstances() {
	c.instances.With(prometheus.Labels{
		"resource": "managed_instances",
	}).Dec()
}

func (c *collector) AddEvent(resource, operation string) {
	c.controllerEvents.With(prometheus.Labels{
		"operation": operation,
		"resource":  resource,
	}).Inc()
}

// Collect collects the prometheus.Collect.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.instances.Collect(ch)
	c.controllerEvents.Collect(ch)
}

// Describe describes the prometheus.Describe.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	c.instances.Describe(ch)
	c.controllerEvents.Describe(ch)
}
