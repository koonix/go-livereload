// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

package pubsub_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/koonix/go-livereload/internal/pubsub"
)

func TestPubSub(t *testing.T) {

	ps := pubsub.New[string]()
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
		fmt.Println("1")
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
		fmt.Println("2")
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
		fmt.Println("3")
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
