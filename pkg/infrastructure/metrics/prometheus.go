package metrics

import (
	"fmt"
	"sync"
)

// PrometheusMetrics implements Metrics interface with Prometheus-compatible format
type PrometheusMetrics struct {
	counters   map[string]*PrometheusCounter
	histograms map[string]*PrometheusHistogram
	gauges     map[string]*PrometheusGauge
	mu         sync.RWMutex
}

// PrometheusCounter implements Counter for Prometheus
type PrometheusCounter struct {
	name   string
	value  float64
	labels map[string]string
	mu     sync.Mutex
}

// PrometheusHistogram implements Histogram for Prometheus
type PrometheusHistogram struct {
	name    string
	buckets []float64
	values  []float64
	labels  map[string]string
	mu      sync.Mutex
}

// PrometheusGauge implements Gauge for Prometheus
type PrometheusGauge struct {
	name   string
	value  float64
	labels map[string]string
	mu     sync.Mutex
}

// NewPrometheusMetrics creates a new Prometheus metrics collector
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		counters:   make(map[string]*PrometheusCounter),
		histograms: make(map[string]*PrometheusHistogram),
		gauges:     make(map[string]*PrometheusGauge),
	}
}

// Counter returns or creates a counter metric
func (p *PrometheusMetrics) Counter(name string) Counter {
	p.mu.Lock()
	defer p.mu.Unlock()

	if counter, exists := p.counters[name]; exists {
		return counter
	}

	counter := &PrometheusCounter{
		name:   name,
		value:  0,
		labels: make(map[string]string),
	}
	p.counters[name] = counter
	return counter
}

// Histogram returns or creates a histogram metric
func (p *PrometheusMetrics) Histogram(name string) Histogram {
	p.mu.Lock()
	defer p.mu.Unlock()

	if histogram, exists := p.histograms[name]; exists {
		return histogram
	}

	histogram := &PrometheusHistogram{
		name:    name,
		buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		values:  make([]float64, 0),
		labels:  make(map[string]string),
	}
	p.histograms[name] = histogram
	return histogram
}

// Gauge returns or creates a gauge metric
func (p *PrometheusMetrics) Gauge(name string) Gauge {
	p.mu.Lock()
	defer p.mu.Unlock()

	if gauge, exists := p.gauges[name]; exists {
		return gauge
	}

	gauge := &PrometheusGauge{
		name:   name,
		value:  0,
		labels: make(map[string]string),
	}
	p.gauges[name] = gauge
	return gauge
}

// Counter methods

func (c *PrometheusCounter) Inc() {
	c.Add(1)
}

func (c *PrometheusCounter) Add(delta float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += delta
}

func (c *PrometheusCounter) WithLabels(labels map[string]string) Counter {
	c.mu.Lock()
	defer c.mu.Unlock()

	newLabels := make(map[string]string)
	for k, v := range c.labels {
		newLabels[k] = v
	}
	for k, v := range labels {
		newLabels[k] = v
	}

	return &PrometheusCounter{
		name:   c.name,
		value:  c.value,
		labels: newLabels,
	}
}

// Histogram methods

func (h *PrometheusHistogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = append(h.values, value)
}

func (h *PrometheusHistogram) WithLabels(labels map[string]string) Histogram {
	h.mu.Lock()
	defer h.mu.Unlock()

	newLabels := make(map[string]string)
	for k, v := range h.labels {
		newLabels[k] = v
	}
	for k, v := range labels {
		newLabels[k] = v
	}

	return &PrometheusHistogram{
		name:    h.name,
		buckets: h.buckets,
		values:  h.values,
		labels:  newLabels,
	}
}

// Gauge methods

func (g *PrometheusGauge) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = value
}

func (g *PrometheusGauge) Inc() {
	g.Add(1)
}

func (g *PrometheusGauge) Dec() {
	g.Sub(1)
}

func (g *PrometheusGauge) Add(delta float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value += delta
}

func (g *PrometheusGauge) Sub(delta float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value -= delta
}

func (g *PrometheusGauge) WithLabels(labels map[string]string) Gauge {
	g.mu.Lock()
	defer g.mu.Unlock()

	newLabels := make(map[string]string)
	for k, v := range g.labels {
		newLabels[k] = v
	}
	for k, v := range labels {
		newLabels[k] = v
	}

	return &PrometheusGauge{
		name:   g.name,
		value:  g.value,
		labels: newLabels,
	}
}

// Export formats metrics in Prometheus text format
func (p *PrometheusMetrics) Export() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var output string

	// Export counters
	for name, counter := range p.counters {
		counter.mu.Lock()
		labelStr := formatLabels(counter.labels)
		output += fmt.Sprintf("# TYPE %s counter\n", name)
		output += fmt.Sprintf("%s%s %v\n", name, labelStr, counter.value)
		counter.mu.Unlock()
	}

	// Export histograms
	for name, histogram := range p.histograms {
		histogram.mu.Lock()
		labelStr := formatLabels(histogram.labels)
		output += fmt.Sprintf("# TYPE %s histogram\n", name)

		// Calculate bucket counts
		bucketCounts := make(map[float64]int)
		for _, value := range histogram.values {
			for _, bucket := range histogram.buckets {
				if value <= bucket {
					bucketCounts[bucket]++
				}
			}
			bucketCounts[+Inf]++
		}

		for _, bucket := range histogram.buckets {
			count := bucketCounts[bucket]
			output += fmt.Sprintf("%s_bucket{le=\"%v\"}%s %d\n", name, bucket, labelStr, count)
		}
		output += fmt.Sprintf("%s_bucket{le=\"+Inf\"}%s %d\n", name, labelStr, len(histogram.values))
		output += fmt.Sprintf("%s_sum%s %v\n", name, labelStr, sum(histogram.values))
		output += fmt.Sprintf("%s_count%s %d\n", name, labelStr, len(histogram.values))
		histogram.mu.Unlock()
	}

	// Export gauges
	for name, gauge := range p.gauges {
		gauge.mu.Lock()
		labelStr := formatLabels(gauge.labels)
		output += fmt.Sprintf("# TYPE %s gauge\n", name)
		output += fmt.Sprintf("%s%s %v\n", name, labelStr, gauge.value)
		gauge.mu.Unlock()
	}

	return output
}

// formatLabels formats labels for Prometheus output
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%q", k, v))
	}
	if len(parts) == 0 {
		return ""
	}

	// Join all parts with comma
	result := "{" + parts[0]
	for i := 1; i < len(parts); i++ {
		result += "," + parts[i]
	}
	result += "}"
	return result
}

// sum calculates the sum of float64 values
func sum(values []float64) float64 {
	var total float64
	for _, v := range values {
		total += v
	}
	return total
}

const Inf = 1e308
