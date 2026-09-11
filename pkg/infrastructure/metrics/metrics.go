package metrics

// Counter represents a counter metric
type Counter interface {
	Inc()
	Add(delta float64)
	WithLabels(labels map[string]string) Counter
}

// Histogram represents a histogram metric
type Histogram interface {
	Observe(value float64)
	WithLabels(labels map[string]string) Histogram
}

// Gauge represents a gauge metric
type Gauge interface {
	Set(value float64)
	Inc()
	Dec()
	Add(delta float64)
	Sub(delta float64)
	WithLabels(labels map[string]string) Gauge
}

// Metrics is the main interface for metrics collection
type Metrics interface {
	Counter(name string) Counter
	Histogram(name string) Histogram
	Gauge(name string) Gauge
}

// NoOpMetrics is a no-op implementation for when metrics are disabled
type NoOpMetrics struct{}

// Counter returns a no-op counter
func (n *NoOpMetrics) Counter(name string) Counter {
	return &NoOpCounter{}
}

// Histogram returns a no-op histogram
func (n *NoOpMetrics) Histogram(name string) Histogram {
	return &NoOpHistogram{}
}

// Gauge returns a no-op gauge
func (n *NoOpMetrics) Gauge(name string) Gauge {
	return &NoOpGauge{}
}

// NoOpCounter is a no-op counter implementation
type NoOpCounter struct{}

func (n *NoOpCounter) Inc()                                        {}
func (n *NoOpCounter) Add(delta float64)                           {}
func (n *NoOpCounter) WithLabels(labels map[string]string) Counter { return n }

// NoOpHistogram is a no-op histogram implementation
type NoOpHistogram struct{}

func (n *NoOpHistogram) Observe(value float64)                         {}
func (n *NoOpHistogram) WithLabels(labels map[string]string) Histogram { return n }

// NoOpGauge is a no-op gauge implementation
type NoOpGauge struct{}

func (n *NoOpGauge) Set(value float64)                         {}
func (n *NoOpGauge) Inc()                                      {}
func (n *NoOpGauge) Dec()                                      {}
func (n *NoOpGauge) Add(delta float64)                         {}
func (n *NoOpGauge) Sub(delta float64)                         {}
func (n *NoOpGauge) WithLabels(labels map[string]string) Gauge { return n }
