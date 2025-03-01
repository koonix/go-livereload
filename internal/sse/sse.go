// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

// Package sse provides an [http.Handler] that implements [Server-Sent Events].
//
// [Server-Sent Events]: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events
package sse

import (
	"fmt"
	"net/http"
	"time"

	"github.com/koonix/go-livereload/internal/pubsub"
)

// Handler is an [http.Handler] that implements [Server-Sent Events].
//
// [Server-Sent Events]: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events
type Handler struct {
	pubsub *pubsub.PubSub[string]
}

func New() *Handler {
	return &Handler{
		pubsub: pubsub.New[string](),
	}
}

func (h *Handler) Publish(eventType, data string) {
	h.pubsub.Publish(event(eventType, data))
}

func (h *Handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	flusher, ok := resp.(http.Flusher)
	if !ok {
		msg := "ability to flush responses unavailable"
		http.Error(resp, msg, http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "text/event-stream")
	resp.Header().Set("Cache-Control", "no-store")
	resp.Header().Set("Connection", "keep-alive")
	resp.Header().Set("X-Accel-Buffering", "no")
	resp.WriteHeader(http.StatusOK)
	flusher.Flush()

	evChan, unsub := h.pubsub.Subscribe()
	defer unsub()

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	for {
		select {

		case <-req.Context().Done():
			return

		case ev := <-evChan:
			_, err := resp.Write([]byte(ev))
			if err != nil {
				return
			}
			flusher.Flush()

		case <-t.C:
			_, err := resp.Write([]byte(event("message", "ping")))
			if err != nil {
				return
			}
			flusher.Flush()

		}
	}
}

func event(eventType, data string) string {
	return fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data)
}
