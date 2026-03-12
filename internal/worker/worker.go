// Package worker contain a generic worker pool struct that can be used for
// several services
package worker

import (
	"context"
	"log"
)

type Pool[T any] struct {
	jobs    chan T
	handler func(T)
	done    chan struct{}
}

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

func (p *Pool[T]) Send(job T) bool {
	select {
	case p.jobs <- job:
		return true
	default:
		return false
	}
}

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
