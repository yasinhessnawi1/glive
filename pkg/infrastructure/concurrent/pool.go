package concurrent

import (
	"context"
	"sync"
)

// WorkerPool manages a pool of workers for parallel tasks
type WorkerPool struct {
	workers   int
	taskQueue chan func()
	wg        sync.WaitGroup
}

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(workers int) *WorkerPool {
	p := &WorkerPool{
		workers:   workers,
		taskQueue: make(chan func(), workers*10),
	}

	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	return p
}

// worker processes tasks from the queue
func (p *WorkerPool) worker() {
	defer p.wg.Done()
	for task := range p.taskQueue {
		task()
	}
}

// Submit submits a task to the worker pool
func (p *WorkerPool) Submit(task func()) {
	p.wg.Add(1)
	p.taskQueue <- func() {
		defer p.wg.Done()
		task()
	}
}

// Wait waits for all submitted tasks to complete
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

// Close closes the task queue and waits for workers to finish
func (p *WorkerPool) Close() {
	close(p.taskQueue)
	p.wg.Wait()
}

// ParallelScanner scans files in parallel
type ParallelScanner struct {
	pool       *WorkerPool
	maxWorkers int
}

// NewParallelScanner creates a new parallel scanner
func NewParallelScanner(maxWorkers int) *ParallelScanner {
	return &ParallelScanner{
		pool:       NewWorkerPool(maxWorkers),
		maxWorkers: maxWorkers,
	}
}

// ScanFiles scans multiple files in parallel using the provided scan function
func (s *ParallelScanner) ScanFiles(ctx context.Context, files []string, scanFunc func(string) error) []error {
	var mu sync.Mutex
	var errors []error

	for _, file := range files {
		file := file // capture loop variable
		s.pool.Submit(func() {
			select {
			case <-ctx.Done():
				return
			default:
				if err := scanFunc(file); err != nil {
					mu.Lock()
					errors = append(errors, err)
					mu.Unlock()
				}
			}
		})
	}

	s.pool.Wait()
	return errors
}

// Close closes the parallel scanner and its worker pool
func (s *ParallelScanner) Close() {
	s.pool.Close()
}

// Task represents a task that can be executed by a worker pool
type Task interface {
	Execute() Result
}

// Result represents the result of executing a task
type Result struct {
	Value interface{}
	Error error
}

// Pool manages a pool of workers for executing tasks
// It matches the specification pattern with Task and Result types
type Pool struct {
	workers int
	tasks   chan Task
	results chan Result
	wg      sync.WaitGroup
}

// NewPool creates a new worker pool with the specified number of workers
func NewPool(workers int) *Pool {
	return &Pool{
		workers: workers,
		tasks:   make(chan Task),
		results: make(chan Result),
	}
}

// Start launches the worker goroutines
func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// worker processes tasks from the queue
func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		result := task.Execute()
		p.results <- result
	}
}

// Submit submits a task to the worker pool
func (p *Pool) Submit(task Task) {
	p.tasks <- task
}

// Results returns the results channel
func (p *Pool) Results() <-chan Result {
	return p.results
}

// Close closes the task queue and waits for workers to finish
func (p *Pool) Close() {
	close(p.tasks)
	p.wg.Wait()
	close(p.results)
}
