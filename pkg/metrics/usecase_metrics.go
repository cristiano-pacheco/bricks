package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	usecaseDurationMetricName = "usecase_duration_seconds"
	usecaseSuccessMetricName  = "usecase_success_total"
	usecaseErrorMetricName    = "usecase_error_total"
)

type UseCaseMetrics interface {
	ObserveDuration(name string, duration time.Duration)
	IncSuccess(name string)
	IncError(name string)
}

var durationBuckets = []float64{0.005, 0.025, 0.05, 0.1, 0.15, 0.2, 0.3, 0.4, 0.5, 1, 1.5, 2, 3, 4, 5}

type PrometheusUseCaseMetrics struct {
	duration *prometheus.HistogramVec
	success  *prometheus.CounterVec
	error    *prometheus.CounterVec
}

var _ UseCaseMetrics = &PrometheusUseCaseMetrics{}

func NewPrometheusUseCaseMetrics() (*PrometheusUseCaseMetrics, error) {
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    usecaseDurationMetricName,
			Help:    "Duration of use case execution in seconds",
			Buckets: durationBuckets,
		},
		[]string{"name"},
	)
	successCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: usecaseSuccessMetricName,
			Help: "Total successful use case executions",
		},
		[]string{"name"},
	)
	errorCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: usecaseErrorMetricName,
			Help: "Total failed use case executions",
		},
		[]string{"name"},
	)

	if err := prometheus.Register(duration); err != nil {
		return nil, err
	}
	if err := prometheus.Register(successCounter); err != nil {
		return nil, err
	}
	if err := prometheus.Register(errorCounter); err != nil {
		return nil, err
	}

	return &PrometheusUseCaseMetrics{
		duration: duration,
		success:  successCounter,
		error:    errorCounter,
	}, nil
}

func (p *PrometheusUseCaseMetrics) ObserveDuration(name string, duration time.Duration) {
	p.duration.WithLabelValues(name).Observe(duration.Seconds())
}

func (p *PrometheusUseCaseMetrics) IncSuccess(name string) {
	p.success.WithLabelValues(name).Inc()
}

func (p *PrometheusUseCaseMetrics) IncError(name string) {
	p.error.WithLabelValues(name).Inc()
}
