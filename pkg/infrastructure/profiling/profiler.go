package profiling

import (
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

// Profiler manages CPU and memory profiling
type Profiler struct {
	cpuFile   *os.File
	memFile   *os.File
	traceFile *os.File
	startTime time.Time
}

// NewProfiler creates a new profiler with the given prefix for output files
func NewProfiler(prefix string) (*Profiler, error) {
	p := &Profiler{startTime: time.Now()}

	var err error
	p.cpuFile, err = os.Create(prefix + "_cpu.prof")
	if err != nil {
		return nil, err
	}

	p.memFile, err = os.Create(prefix + "_mem.prof")
	if err != nil {
		p.cpuFile.Close()
		return nil, err
	}

	return p, nil
}

// Start begins CPU profiling
func (p *Profiler) Start() error {
	return pprof.StartCPUProfile(p.cpuFile)
}

// Stop stops profiling and writes memory profile
func (p *Profiler) Stop() error {
	pprof.StopCPUProfile()
	if err := p.cpuFile.Close(); err != nil {
		return err
	}

	runtime.GC() // Get accurate memory stats
	if err := pprof.WriteHeapProfile(p.memFile); err != nil {
		return err
	}
	if err := p.memFile.Close(); err != nil {
		return err
	}

	return nil
}

// Metrics tracks runtime metrics
type Metrics struct {
	StartTime       time.Time
	Allocations     uint64
	TotalAlloc      uint64
	HeapInUse       uint64
	NumGC           uint32
	GCPauseTotal    time.Duration
}

// CaptureMetrics captures current runtime metrics
func CaptureMetrics() *Metrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &Metrics{
		StartTime:    time.Now(),
		Allocations:  m.Mallocs,
		TotalAlloc:   m.TotalAlloc,
		HeapInUse:    m.HeapInuse,
		NumGC:        m.NumGC,
		GCPauseTotal: time.Duration(m.PauseTotalNs),
	}
}

// Delta calculates the difference between two metrics snapshots
func (m *Metrics) Delta(other *Metrics) *Metrics {
	return &Metrics{
		Allocations:  m.Allocations - other.Allocations,
		TotalAlloc:   m.TotalAlloc - other.TotalAlloc,
		HeapInUse:   m.HeapInUse,
		NumGC:       m.NumGC - other.NumGC,
		GCPauseTotal: m.GCPauseTotal - other.GCPauseTotal,
	}
}

