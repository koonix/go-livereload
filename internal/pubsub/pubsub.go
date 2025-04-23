// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

// Package pubsub provides publish-subscribe functionality
// that's designed to scale down to zero subscribers
// without leaking any goroutines.
package pubsub

import (
	"runtime"
	"sync"
)

type PubSub[T any] struct {
	msg       chan T
	addSub    chan *sub[T]
	removeSub chan *sub[T]
	done      chan struct{}
	once      sync.Once
}

type sub[T any] struct {
	msg  chan T
	done chan struct{}
	once sync.Once
}

func New[T any]() *PubSub[T] {

	p := &PubSub[T]{
		msg:       make(chan T),
		addSub:    make(chan *sub[T]),
		removeSub: make(chan *sub[T]),
		done:      make(chan struct{}),
	}
	runtime.SetFinalizer(p, func(p *PubSub[T]) {
		p.Close()
	})

	msg := p.msg
	addSub := p.addSub
	removeSub := p.removeSub
	done := p.done

	go func() {
		subs := make(map[*sub[T]]struct{})
		defer func() {
			for sub := range subs {
				close(sub.msg)
			}
		}()
		for {
			select {
			case <-done:
				return
			case sub := <-addSub:
				subs[sub] = struct{}{}
			case sub := <-removeSub:
				delete(subs, sub)
				close(sub.msg)
			case msg := <-msg:
				wg := new(sync.WaitGroup)
				wg.Add(len(subs))
				for sub := range subs {
					go func() {
						defer wg.Done()
						select {
						case sub.msg <- msg:
						case <-sub.done:
						}
					}()
				}
				wg.Wait()
			}
		}
	}()

	return p
}

func (p *PubSub[T]) Subscribe() (msg <-chan T, unsubscribe func()) {

	sub := &sub[T]{
		msg:  make(chan T, 1),
		done: make(chan struct{}),
	}

	select {
	case p.addSub <- sub:
	case <-p.done:
	}

	unsub := func() {
		sub.once.Do(func() {
			close(sub.done)
			select {
			case p.removeSub <- sub:
			case <-p.done:
			}
		})
	}

	return sub.msg, unsub
}

func (p *PubSub[T]) Publish(msg T) {
	select {
	case p.msg <- msg:
	case <-p.done:
	}
}

func (p *PubSub[T]) Close() {
	p.once.Do(func() {
		close(p.done)
	})
}
