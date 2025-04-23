// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

package pubsub

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPubSub(t *testing.T) {

	ps := New[string]()
	wg := new(sync.WaitGroup)
	rcvCount := new(atomic.Int64)
	msg := "hello world"

	ch1, unsub1 := ps.Subscribe()
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer unsub1()
		for m := range ch1 {
			if m == msg {
				rcvCount.Add(1)
			} else {
				t.Errorf("ch1 got incorrect message; want %q, got %q", msg, m)
			}
		}
	}()

	ch2, unsub2 := ps.Subscribe()
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer unsub2()
		for m := range ch2 {
			if m == msg {
				rcvCount.Add(1)
			} else {
				t.Errorf("ch2 got incorrect message; want %q, got %q", msg, m)
			}
		}
	}()

	ch3, unsub3 := ps.Subscribe()
	unsub3() // Unsubscribe immediately.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer unsub3()
		for range ch3 {
			t.Errorf("ch3 got message, didn't expect any")
		}
	}()

	ps.Publish(msg)
	ps.Close()
	wg.Wait()

	want := 2
	got := int(rcvCount.Load())
	if want != got {
		t.Errorf("incorrect message count; want %d, got %d", want, got)
	}
}

func TestPubSubFinalizer(t *testing.T) {

	// Create the PubSub inside an inner scope
	// so the pointer becomes unreachable when we leave the block.
	done := func() <-chan struct{} {
		p := New[int]()
		return p.done
	}()

	runtime.GC()

	select {
	case <-done:
		// Success.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("finalizer did not run")
	}
}
