package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// YcMetrics describes metrics types
type YcMetrics struct {
	APIDuration *prometheus.HistogramVec
	APIRequests *prometheus.CounterVec
}

// MetricContext indicates the context for Yandex Cloud API metrics.
type MetricContext struct {
	Start      time.Time
	Attributes []string
	Metrics    *YcMetrics
}

// Metrics labels
const (
	metricYCSubsystem           string = "yc" // yandexcloud or capy
	metricRequestCountKey       string = "api_requests"
	metricRequestDurationKey    string = "api_request_duration_seconds"
	metricServiceLabel          string = "object"     // compute, loadbalancer
	metricControllerLabel       string = "controller" // yandexmachine, yandexcluster
	metricStatusLabel           string = "status"
	StatusFailed                string = "failed"
	StatusSuccess               string = "success"
	ServiceLabelCompute         string = "compute"
	ServiceLabelAlbTargetGroup  string = "alb-target-group"
	ServiceLabelAlbBackendGroup string = "alb-backend-group"
	ServiceLabelAlb             string = "alb"
	ServiceLabelNlbTargetGroup  string = "nlb-target-group"
	ControllerLabelMachine      string = "yandexmachine"
)

var durationBuckets = []float64{0, .1, .25, .5, .75, 1., 5.}

var apiRequestMetrics = &YcMetrics{
	APIDuration: prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: metricYCSubsystem,
			Name:      metricRequestDurationKey,
			Help:      "Latency of HTTP requests to YandexCloud API",
			Buckets:   durationBuckets,
		}, []string{metricControllerLabel, metricServiceLabel}),
	APIRequests: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: metricYCSubsystem,
			Name:      metricRequestCountKey,
			Help:      "Number of YandexCloud API requests",
		}, []string{metricControllerLabel, metricServiceLabel, metricStatusLabel}),
}

var registerAPIMetrics sync.Once

// RegisterAPIMetrics registers metrics
func RegisterAPIMetrics() {
	registerAPIMetrics.Do(func() {
		metrics.Registry.MustRegister(apiRequestMetrics.APIDuration)
		metrics.Registry.MustRegister(apiRequestMetrics.APIRequests)
	})
}

// NewMetricContext creates a new MetricContext.
func NewMetricContext(controller, service string) *MetricContext {
	return &MetricContext{
		Start:      time.Now(),
		Attributes: []string{controller, service},
	}
}

// ObserveRequest records the request latency and counts errors.
func (mc *MetricContext) ObserveRequest(err error) {
	mc.observe(apiRequestMetrics, err)
}

// observe records the request latency and counts errors.
func (mc *MetricContext) observe(om *YcMetrics, err error) {
	om.APIDuration.WithLabelValues(mc.Attributes...).Observe(
		time.Since(mc.Start).Seconds())
	if err != nil {
		om.APIRequests.WithLabelValues(append(mc.Attributes, StatusFailed)...).Inc()
		return
	}
	om.APIRequests.WithLabelValues(append(mc.Attributes, StatusSuccess)...).Inc()
}
