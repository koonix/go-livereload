// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

// Package resprouter provides an [http.ResponseWriter]
// that allows deciding where the response body is written to
// based on the response header.
package resprouter

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"
)

type Router struct {
	StatusCode    int
	SniffSize     int
	SniffDuration time.Duration
	Done          chan io.Writer

	headerRouter HeaderRouter
	sniffRouter  SniffRouter

	header      http.Header
	wroteHeader bool

	mu     sync.Mutex
	writer io.Writer
}

type (
	HeaderRouter func(r *Router) io.Writer
	SniffRouter  func(r *Router, sniffed []byte) io.Writer
)

// ==========

func New(h HeaderRouter, s SniffRouter) *Router {
	return &Router{
		SniffSize:     512,
		SniffDuration: 100 * time.Millisecond,
		Done:          make(chan io.Writer, 1),
		headerRouter:  h,
		sniffRouter:   s,
		header:        make(http.Header),
	}
}

func (r *Router) Header() http.Header {
	return r.header
}

func (r *Router) Write(data []byte) (n int, err error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.writer.Write(data)
}

func (r *Router) WriteHeader(statusCode int) {

	if r.wroteHeader {
		return
	}

	r.wroteHeader = true
	r.StatusCode = statusCode

	w := r.headerRouter(r)
	if w != nil {
		r.writer = w
		r.Done <- w
		close(r.Done)
		return
	}

	buf := new(bytes.Buffer)
	sniffed := false
	runSniffRouter := func() {
		if sniffed {
			return
		}
		sniffed = true
		data := buf.Bytes()
		w := r.sniffRouter(r, data)
		w.Write(data)
		r.writer = w
		r.Done <- w
		close(r.Done)
	}

	var t *time.Timer

	if r.SniffDuration > 0 {
		t = time.AfterFunc(r.SniffDuration, func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			runSniffRouter()
		})
	}

	r.writer = &writer{fn: func(data []byte) (n int, err error) {
		n, err = buf.Write(data)
		if r.SniffSize > 0 && buf.Len() >= r.SniffSize {
			runSniffRouter()
			if t != nil {
				t.Stop()
			}
		}
		return
	}}
}

// ==========

type writer struct {
	fn func(data []byte) (int, error)
}

func (w *writer) Write(data []byte) (int, error) {
	return w.fn(data)
}

// ==========

func CopyHeader(src http.Header, destinations ...http.Header) {
	for key, values := range src {
		for _, dst := range destinations {
			dst.Del(key)
			for _, v := range values {
				dst.Add(key, v)
			}
		}
	}
}
