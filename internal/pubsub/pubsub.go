// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

// Package pubsub provides publish-subscribe functionality
// that's designed to scale down to zero subscribers
// without leaking any goroutines.
package pubsub

import (
	"sync"
)

type PubSub[T any] struct {
	mu          sync.Mutex
	subscribers map[*subscriber[T]]struct{}
}

type subscriber[T any] struct {
	msg    chan T
	done   chan struct{}
	closed bool
}

func New[T any]() *PubSub[T] {
	return &PubSub[T]{
		subscribers: make(map[*subscriber[T]]struct{}),
	}
}

func (p *PubSub[T]) Publish(msg T) {

	p.mu.Lock()
	defer p.mu.Unlock()

	wg := new(sync.WaitGroup)
	wg.Add(len(p.subscribers))

	for s := range p.subscribers {
		go func() {
			defer wg.Done()
			select {
			case s.msg <- msg:
			case <-s.done:
			}
		}()
	}

	wg.Wait()
}

func (p *PubSub[T]) Subscribe() (msg <-chan T, unsubscribe func()) {

	s := &subscriber[T]{
		msg:  make(chan T),
		done: make(chan struct{}),
	}

	p.mu.Lock()
	p.subscribers[s] = struct{}{}
	p.mu.Unlock()

	unsub := func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if !s.closed {
			close(s.done)
			s.closed = true
		}
	}

	return s.msg, unsub
}

// Clear unsubscribes all existing subscribers.
func (p *PubSub[T]) Clear() {

	p.mu.Lock()
	defer p.mu.Unlock()

	for s := range p.subscribers {
		if !s.closed {
			close(s.done)
			s.closed = true
		}
	}
}
