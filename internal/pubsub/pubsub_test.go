// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

package pubsub_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/koonix/go-livereload/internal/pubsub"
)

func TestPubSub(t *testing.T) {

	p := pubsub.New[string]()
	wg := new(sync.WaitGroup)
	n := new(atomic.Int64)
	pmsg := "hello world"

	ch1, unsub1 := p.Subscribe()
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer unsub1()
		for msg := range ch1 {
			if msg == pmsg {
				n.Add(1)
			} else {
				t.Errorf("ch1 got incorrect message; want %q, got %q", pmsg, msg)
			}
		}
	}()

	ch2, unsub2 := p.Subscribe()
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer unsub2()
		for msg := range ch2 {
			if msg == pmsg {
				n.Add(1)
			} else {
				t.Errorf("ch2 got incorrect message; want %q, got %q", pmsg, msg)
			}
		}
	}()

	ch3, unsub3 := p.Subscribe()
	unsub3() // Unsubscribe immediately.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer unsub3()
		for range ch3 {
			t.Errorf("ch3 got message, didn't expect any")
		}
	}()

	p.Publish(pmsg)
	p.Clear()
	wg.Wait()

	want := 2
	got := int(n.Load())
	if want != got {
		t.Errorf("incorrect message count; want %d, got %d", want, got)
	}
}
