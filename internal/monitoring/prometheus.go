package monitoring

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"context"
)

// PrometheusClientImpl implements the PrometheusClient interface
type PrometheusClientImpl struct {
	registry   *prometheus.Registry
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	queryAPI   v1.API
}

// NewPrometheusClient creates a new Prometheus client
func NewPrometheusClient(prometheusURL string) (*PrometheusClientImpl, error) {
	registry := prometheus.NewRegistry()
	
	var queryAPI v1.API
	if prometheusURL != "" {
		client, err := api.NewClient(api.Config{
			Address: prometheusURL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
		}
		queryAPI = v1.NewAPI(client)
	}
	
	return &PrometheusClientImpl{
		registry:   registry,
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		queryAPI:   queryAPI,
	}, nil
}

// RecordCounter records a counter metric
func (p *PrometheusClientImpl) RecordCounter(name string, labels map[string]string, value float64) error {
	counter, exists := p.counters[name]
	if !exists {
		// Create new counter
		labelNames := make([]string, 0, len(labels))
		for labelName := range labels {
			labelNames = append(labelNames, labelName)
		}
		
		counter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: name,
				Help: fmt.Sprintf("Counter metric for %s", name),
			},
			labelNames,
		)
		
		if err := p.registry.Register(counter); err != nil {
			// If already registered, get the existing one
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				counter = are.ExistingCollector.(*prometheus.CounterVec)
			} else {
				return fmt.Errorf("failed to register counter %s: %w", name, err)
			}
		}
		
		p.counters[name] = counter
	}
	
	counter.With(labels).Add(value)
	return nil
}

// RecordGauge records a gauge metric
func (p *PrometheusClientImpl) RecordGauge(name string, labels map[string]string, value float64) error {
	gauge, exists := p.gauges[name]
	if !exists {
		// Create new gauge
		labelNames := make([]string, 0, len(labels))
		for labelName := range labels {
			labelNames = append(labelNames, labelName)
		}
		
		gauge = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: name,
				Help: fmt.Sprintf("Gauge metric for %s", name),
			},
			labelNames,
		)
		
		if err := p.registry.Register(gauge); err != nil {
			// If already registered, get the existing one
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				gauge = are.ExistingCollector.(*prometheus.GaugeVec)
			} else {
				return fmt.Errorf("failed to register gauge %s: %w", name, err)
			}
		}
		
		p.gauges[name] = gauge
	}
	
	gauge.With(labels).Set(value)
	return nil
}

// RecordHistogram records a histogram metric
func (p *PrometheusClientImpl) RecordHistogram(name string, labels map[string]string, value float64) error {
	histogram, exists := p.histograms[name]
	if !exists {
		// Create new histogram
		labelNames := make([]string, 0, len(labels))
		for labelName := range labels {
			labelNames = append(labelNames, labelName)
		}
		
		histogram = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: name,
				Help: fmt.Sprintf("Histogram metric for %s", name),
				Buckets: prometheus.DefBuckets,
			},
			labelNames,
		)
		
		if err := p.registry.Register(histogram); err != nil {
			// If already registered, get the existing one
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				histogram = are.ExistingCollector.(*prometheus.HistogramVec)
			} else {
				return fmt.Errorf("failed to register histogram %s: %w", name, err)
			}
		}
		
		p.histograms[name] = histogram
	}
	
	histogram.With(labels).Observe(value)
	return nil
}

// Query executes a Prometheus query
func (p *PrometheusClientImpl) Query(query string) (float64, error) {
	if p.queryAPI == nil {
		return 0, fmt.Errorf("Prometheus query API not configured")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	result, warnings, err := p.queryAPI.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}
	
	if len(warnings) > 0 {
		// Log warnings but continue
		fmt.Printf("Prometheus query warnings: %v\n", warnings)
	}
	
	// Extract scalar value from result
	// This is a simplified implementation - in practice you'd handle different result types
	switch result.Type() {
	case "scalar":
		// Handle scalar result
		return 0, fmt.Errorf("scalar result handling not implemented")
	case "vector":
		// Handle vector result - return first value
		return 0, fmt.Errorf("vector result handling not implemented")
	default:
		return 0, fmt.Errorf("unsupported result type: %s", result.Type())
	}
}

// GetHandler returns the HTTP handler for Prometheus metrics
func (p *PrometheusClientImpl) GetHandler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}

// GetRegistry returns the Prometheus registry
func (p *PrometheusClientImpl) GetRegistry() *prometheus.Registry {
	return p.registry
}