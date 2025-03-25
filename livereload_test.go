// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

package livereload_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/koonix/go-livereload"
)

func Example_fileServer() {
	lr := livereload.New(http.FileServer(http.Dir("frontend")))
	go func() {
		time.Sleep(10 * time.Second)
		lr.Reload()
	}()
	http.ListenAndServe(":8090", lr)
}

func Example_reverseProxy() {
	u, _ := url.Parse("http://localhost:8080")
	lr := livereload.New(livereload.ReverseProxy(u))
	go func() {
		time.Sleep(10 * time.Second)
		lr.Reload()
	}()
	http.ListenAndServe(":8090", lr)
}

func TestLiveReload(t *testing.T) {

	script := []byte("new EventSource")
	content := []byte("plain text body")
	htmlContent := []byte("<p>html body</p>")

	t.Run("no-content-type-plaintext", func(t *testing.T) {
		upstream := &handler{
			Body: content,
		}
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		livereload.New(upstream).ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if !bytes.Contains(body, content) {
			t.Errorf("response does not contain the body")
		}
		if !bytes.Contains(body, script) {
			t.Errorf("response does not contain the event listener script")
		}
		if resp.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("incorrect Cache-Control header")
		}
	})

	t.Run("no-disable-caching", func(t *testing.T) {
		upstream := &handler{
			Body: content,
		}
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		option := livereload.WithDisableCaching(false)
		livereload.New(upstream, option).ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if !bytes.Contains(body, content) {
			t.Errorf("response does not contain the body")
		}
		if !bytes.Contains(body, script) {
			t.Errorf("response does not contain the event listener script")
		}
		if resp.Header().Get("Cache-Control") != "" {
			t.Errorf("incorrect Cache-Control header")
		}
	})

	t.Run("no-content-type-html", func(t *testing.T) {
		upstream := &handler{
			Body: htmlContent,
		}
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		livereload.New(upstream).ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if !bytes.Contains(body, htmlContent) {
			t.Errorf("response does not contain the body")
		}
		if !bytes.Contains(body, script) {
			t.Errorf("response does not contain the event listener script")
		}
	})

	t.Run("content-type-plaintext", func(t *testing.T) {
		upstream := &handler{
			Body:        content,
			ContentType: "text/plain",
		}
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		livereload.New(upstream).ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if !bytes.Contains(body, content) {
			t.Errorf("response does not contain the body")
		}
		if !bytes.Contains(body, script) {
			t.Errorf("response does not contain the event listener script")
		}
	})

	t.Run("content-type-html", func(t *testing.T) {
		upstream := &handler{
			Body:        content,
			ContentType: "text/html",
		}
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		livereload.New(upstream).ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if !bytes.Contains(body, content) {
			t.Errorf("response does not contain the body")
		}
		if !bytes.Contains(body, script) {
			t.Errorf("response does not contain the event listener script")
		}
	})

	t.Run("content-type-other", func(t *testing.T) {
		upstream := &handler{
			Body:        content,
			ContentType: "text/unknown",
		}
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		livereload.New(upstream).ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if !bytes.Equal(body, content) {
			t.Errorf("response of unknown type is modified")
		}
	})

	t.Run("content-disposition-attachment", func(t *testing.T) {
		upstream := &handler{
			Body:               content,
			ContentType:        "text/html",
			ContentDisposition: "attachment",
		}
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		livereload.New(upstream).ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if !bytes.Equal(body, content) {
			t.Errorf("response of unknown type is modified")
		}
	})

	t.Run("bad-request", func(t *testing.T) {
		upstream := &handler{
			Body: content,
		}
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodPut, "/livereloadevents", nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		livereload.New(upstream).ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if !bytes.Contains(body, []byte("method not allowed")) {
			t.Errorf("incorrect response body")
		}
		if resp.Code != http.StatusMethodNotAllowed {
			t.Errorf("incorrect response status code")
		}
	})

	t.Run("no-reload-event", func(t *testing.T) {
		upstream := &handler{
			Body: content,
		}
		resp := httptest.NewRecorder()
		ctx, cancel := context.WithCancel(context.Background())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/livereloadevents", nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		lr := livereload.New(upstream)
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()
		lr.ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if len(body) != 0 {
			t.Errorf("got event where none was expected")
		}
	})

	t.Run("reload-event", func(t *testing.T) {
		upstream := &handler{
			Body: content,
		}
		resp := httptest.NewRecorder()
		ctx, cancel := context.WithCancel(context.Background())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/livereloadevents", nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		lr := livereload.New(upstream)
		go func() {
			time.Sleep(100 * time.Millisecond)
			lr.Reload()
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()
		lr.ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if !bytes.Contains(body, []byte("event: message\ndata: reload\n")) {
			t.Errorf("response does not contain the reload event")
		}
	})

	t.Run("reload-event-post-request", func(t *testing.T) {
		upstream := &handler{
			Body: content,
		}
		resp := httptest.NewRecorder()
		ctx, cancel := context.WithCancel(context.Background())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/livereloadevents", nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		lr := livereload.New(upstream)
		go func() {
			postReq, _ := http.NewRequest(http.MethodPost, "/livereloadevents", nil)
			time.Sleep(100 * time.Millisecond)
			lr.ServeHTTP(httptest.NewRecorder(), postReq)
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()
		lr.ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if !bytes.Contains(body, []byte("event: message\ndata: reload\n")) {
			t.Errorf("response does not contain the reload event")
		}
	})

	t.Run("reload-event-custom-path", func(t *testing.T) {
		eventPath := "/myEventPath"
		upstream := &handler{
			Body: content,
		}
		resp := httptest.NewRecorder()
		ctx, cancel := context.WithCancel(context.Background())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, eventPath, nil)
		if err != nil {
			t.Fatalf("could not create request: %s", err)
		}
		option := livereload.WithEventPath(eventPath)
		lr := livereload.New(upstream, option)
		go func() {
			time.Sleep(100 * time.Millisecond)
			lr.Reload()
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()
		lr.ServeHTTP(resp, req)
		body, _ := io.ReadAll(resp.Result().Body)
		if !bytes.Contains(body, []byte("event: message\ndata: reload\n")) {
			t.Errorf("response does not contain the reload event")
		}
	})
}

type handler struct {
	Body               []byte
	ContentType        string
	ContentDisposition string
}

func (h *handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if h.ContentType != "" {
		resp.Header().Set("Content-Type", h.ContentType)
	}
	if h.ContentDisposition != "" {
		resp.Header().Set("Content-Disposition", h.ContentDisposition)
	}
	resp.Write(h.Body)
}
