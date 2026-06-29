// Package worker provides a generic worker pool that can be used for several services.
package worker

import (
	"context"
	"log"
)

// Pool is a generic worker pool that processes jobs concurrently.
type Pool[T any] struct {
	jobs    chan T
	handler func(T)
	done    chan struct{}
}

// NewPool creates a new worker pool with the specified number of workers and queue size.
func NewPool[T any](workers, queueSize int, handler func(T)) *Pool[T] {
	p := &Pool[T]{
		jobs:    make(chan T, queueSize),
		handler: handler,
		done:    make(chan struct{}),
	}
	for range workers {
		go p.run()
	}
	return p
}

func (p *Pool[T]) run() {
	for job := range p.jobs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("worker panic: %v", r)
				}
			}()
			p.handler(job)
		}()
	}
	p.done <- struct{}{}
}

// Send attempts to send a job to the pool without blocking.
// Returns true if the job was sent successfully, false if the queue is full.
func (p *Pool[T]) Send(job T) bool {
	select {
	case p.jobs <- job:
		return true
	default:
		return false
	}
}

// Shutdown gracefully shuts down the worker pool.
// It closes the job channel and waits for all workers to complete.
// Returns early if the context is cancelled.
func (p *Pool[T]) Shutdown(ctx context.Context) {
	close(p.jobs)
	for range cap(p.done) {
		select {
		case <-p.done:
		case <-ctx.Done():
			return
		}
	}
}
