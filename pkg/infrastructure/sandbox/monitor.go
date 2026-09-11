package sandbox

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ResourceMonitor monitors resource usage during sandbox execution
type ResourceMonitor struct {
	mu          sync.RWMutex
	cpuUsage    float64
	memoryUsed  int64
	diskUsed    int64
	networkSent int64
	networkRecv int64
	thresholds  ResourceThresholds
	alerts      []ResourceAlert
	monitoring  bool
	stopChan    chan struct{}
}

// ResourceThresholds defines thresholds for resource monitoring
type ResourceThresholds struct {
	MaxCPUPercent   float64
	MaxMemoryBytes  int64
	MaxDiskBytes    int64
	MaxNetworkBytes int64
}

// ResourceAlert represents a resource threshold violation
type ResourceAlert struct {
	Resource  string
	Value     float64
	Threshold float64
	Timestamp time.Time
	Severity  string // "warning", "critical"
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(thresholds ResourceThresholds) *ResourceMonitor {
	return &ResourceMonitor{
		thresholds: thresholds,
		alerts:     make([]ResourceAlert, 0),
		stopChan:   make(chan struct{}),
	}
}

// Start begins monitoring resource usage
func (rm *ResourceMonitor) Start(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.monitoring {
		return fmt.Errorf("monitor already running")
	}

	rm.monitoring = true
	rm.stopChan = make(chan struct{})

	go rm.monitorLoop(ctx)

	return nil
}

// Stop stops monitoring
func (rm *ResourceMonitor) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.monitoring {
		return
	}

	rm.monitoring = false
	close(rm.stopChan)
}

// UpdateCPU updates CPU usage
func (rm *ResourceMonitor) UpdateCPU(usage float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.cpuUsage = usage
	rm.checkThreshold("cpu", usage, rm.thresholds.MaxCPUPercent)
}

// UpdateMemory updates memory usage
func (rm *ResourceMonitor) UpdateMemory(used int64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.memoryUsed = used
	if rm.thresholds.MaxMemoryBytes > 0 {
		rm.checkThreshold("memory", float64(used), float64(rm.thresholds.MaxMemoryBytes))
	}
}

// UpdateDisk updates disk usage
func (rm *ResourceMonitor) UpdateDisk(used int64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.diskUsed = used
	if rm.thresholds.MaxDiskBytes > 0 {
		rm.checkThreshold("disk", float64(used), float64(rm.thresholds.MaxDiskBytes))
	}
}

// UpdateNetwork updates network usage
func (rm *ResourceMonitor) UpdateNetwork(sent, recv int64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.networkSent = sent
	rm.networkRecv = recv
	total := sent + recv
	if rm.thresholds.MaxNetworkBytes > 0 {
		rm.checkThreshold("network", float64(total), float64(rm.thresholds.MaxNetworkBytes))
	}
}

// GetMetrics returns current resource metrics
func (rm *ResourceMonitor) GetMetrics() ResourceMetrics {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return ResourceMetrics{
		CPU:         rm.cpuUsage,
		Memory:      rm.memoryUsed,
		Disk:        rm.diskUsed,
		NetworkSent: rm.networkSent,
		NetworkRecv: rm.networkRecv,
	}
}

// GetAlerts returns all resource alerts
func (rm *ResourceMonitor) GetAlerts() []ResourceAlert {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	alerts := make([]ResourceAlert, len(rm.alerts))
	copy(alerts, rm.alerts)
	return alerts
}

// ClearAlerts clears all alerts
func (rm *ResourceMonitor) ClearAlerts() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.alerts = make([]ResourceAlert, 0)
}

// ResourceMetrics represents current resource usage
type ResourceMetrics struct {
	CPU         float64
	Memory      int64
	Disk        int64
	NetworkSent int64
	NetworkRecv int64
}

// monitorLoop runs the monitoring loop
func (rm *ResourceMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rm.stopChan:
			return
		case <-ticker.C:
			// Check thresholds periodically
			rm.checkAllThresholds()
		}
	}
}

// checkAllThresholds checks all resource thresholds
func (rm *ResourceMonitor) checkAllThresholds() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.thresholds.MaxCPUPercent > 0 {
		rm.checkThreshold("cpu", rm.cpuUsage, rm.thresholds.MaxCPUPercent)
	}
	if rm.thresholds.MaxMemoryBytes > 0 {
		rm.checkThreshold("memory", float64(rm.memoryUsed), float64(rm.thresholds.MaxMemoryBytes))
	}
	if rm.thresholds.MaxDiskBytes > 0 {
		rm.checkThreshold("disk", float64(rm.diskUsed), float64(rm.thresholds.MaxDiskBytes))
	}
	if rm.thresholds.MaxNetworkBytes > 0 {
		total := rm.networkSent + rm.networkRecv
		rm.checkThreshold("network", float64(total), float64(rm.thresholds.MaxNetworkBytes))
	}
}

// checkThreshold checks if a threshold is exceeded
func (rm *ResourceMonitor) checkThreshold(resource string, value, threshold float64) {
	if threshold <= 0 {
		return
	}

	percent := (value / threshold) * 100

	severity := "warning"
	if percent >= 100 {
		severity = "critical"
	} else if percent < 80 {
		return // No alert needed
	}

	alert := ResourceAlert{
		Resource:  resource,
		Value:     value,
		Threshold: threshold,
		Timestamp: time.Now(),
		Severity:  severity,
	}

	rm.alerts = append(rm.alerts, alert)

	// Keep only last 100 alerts
	if len(rm.alerts) > 100 {
		rm.alerts = rm.alerts[len(rm.alerts)-100:]
	}
}

// ShouldKill determines if the process should be killed based on thresholds
func (rm *ResourceMonitor) ShouldKill() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Check for critical alerts
	for _, alert := range rm.alerts {
		if alert.Severity == "critical" {
			return true
		}
	}

	// Check current values exceed thresholds
	if rm.thresholds.MaxCPUPercent > 0 && rm.cpuUsage >= rm.thresholds.MaxCPUPercent {
		return true
	}
	if rm.thresholds.MaxMemoryBytes > 0 && rm.memoryUsed >= rm.thresholds.MaxMemoryBytes {
		return true
	}
	if rm.thresholds.MaxDiskBytes > 0 && rm.diskUsed >= rm.thresholds.MaxDiskBytes {
		return true
	}

	return false
}
