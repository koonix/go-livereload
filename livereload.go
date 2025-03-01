// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

// Package livereload provides remote webpage reloading functionality,
// by injecting a script into HTML responses that listens to [Server-Sent Events].
//
// Serve a directory:
//
//	lr := livereload.New(http.FileServer(http.Dir("frontend")))
//	http.ListenAndServe(":8090", lr)
//
// Proxy another webserver:
//
//	u, _ := url.Parse("http://localhost:8080")
//	lr := livereload.New(livereload.ReverseProxy(u))
//	http.ListenAndServe(":8090", lr)
//
// Reload:
//
//	lr.Reload()
//
// [Server-Sent Events]: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events
package livereload

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/koonix/go-livereload/internal/htmlpatch"
	"github.com/koonix/go-livereload/internal/resprouter"
	"github.com/koonix/go-livereload/internal/retrier"
	"github.com/koonix/go-livereload/internal/sse"
	"golang.org/x/net/html"
)

// Handler is returned by [New].
type Handler struct {
	upstream       http.Handler
	eventPath      string
	disableCaching bool
	sseHandler     *sse.Handler
	script         string
}

// New creates a [Handler].
//
// Handler proxies the given upstream handler
// to inject an event listener script into the HTML responses.
// Necessary HTML elements are added if missing.
//
// The event listener script reloads the webpage
// upon receiving reload messages,
// which can be sent using [Handler.Reload],
// or by making a POST request to the event path,
// which is "/livereloadevents" by default.
//
// The default event path can be changed using the [WithEventPath] option.
//
// The header "Cache-Control: no-store"
// is included in the responses, to keep browsers from caching them
// and have them reacquire all resources on each reload.
// Use the [WithDisableCaching] option to control this behavior.
func New(upstream http.Handler, options ...Option) *Handler {
	h := &Handler{
		upstream:       upstream,
		eventPath:      "/livereloadevents",
		disableCaching: true,
		sseHandler:     sse.New(),
	}
	for _, fn := range options {
		fn(h)
	}
	h.script = getScript(h.eventPath)
	return h
}

func (h *Handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path != h.eventPath {
		h.injectScript(resp, req)
		return
	}
	if req.Method == http.MethodGet {
		h.sseHandler.ServeHTTP(resp, req)
		return
	}
	if req.Method == http.MethodPost {
		h.Reload()
		return
	}
	msg := fmt.Sprintf("method not allowed: %q", req.Method)
	http.Error(resp, msg, http.StatusMethodNotAllowed)
}

// Reload signals the webpages to reload.
func (h *Handler) Reload() {
	h.sseHandler.Publish("message", "reload")
}

// ==========

func (h *Handler) injectScript(resp http.ResponseWriter, req *http.Request) {

	// Modify the request to indicate we don't accept response compression.
	req.Header.Set("Accept-Encoding", "identity")

	// buf stores the upstream response
	// when we deduce we need to inject a script in it.
	buf := new(bytes.Buffer)

	// Create the upstream response writer.
	uresp := resprouter.New(
		func(uresp *resprouter.Router) (w io.Writer) {
			resprouter.CopyHeader(uresp.Header(), resp.Header())
			if h.disableCaching {
				resp.Header().Set("Cache-Control", "no-store")
			}
			disp, _, _ := mime.ParseMediaType(uresp.Header().Get("Content-Disposition"))
			if disp == "attachment" {
				return resp
			}
			typ, _, _ := mime.ParseMediaType(uresp.Header().Get("Content-Type"))
			if typ == "text/html" || typ == "text/plain" {
				return buf
			} else if typ == "" {
				return nil
			} else {
				return resp
			}
		},
		func(uresp *resprouter.Router, sniffed []byte) io.Writer {
			typ, _, _ := mime.ParseMediaType(http.DetectContentType(sniffed))
			if typ == "text/html" || typ == "text/plain" {
				return buf
			} else {
				return resp
			}
		},
	)

	// Send the request upstream.
	h.upstream.ServeHTTP(uresp, req)

	// Wait for the upstream response to get routed.
	w := <-uresp.Done

	// If the upstream isn't routed to buf,
	// it means we don't want to modify the response
	// and there is nothing to do.
	if w == resp {
		return
	}

	// Inject the script into the response.
	origHtml := buf.Bytes()
	scriptAttrs := scriptNonceAttrs(resp.Header())
	newHtml, err := htmlpatch.InsertScript(origHtml, scriptAttrs, h.script)
	if err != nil {
		if uresp.StatusCode != http.StatusOK {
			resp.WriteHeader(uresp.StatusCode)
			resp.Write(origHtml)
		} else {
			err := fmt.Errorf("could not insert script into HTML: %w", err)
			http.Error(resp, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Send the modified response downstream.
	resp.Header().Del("Content-Length")
	resp.WriteHeader(uresp.StatusCode)
	resp.Write(append(newHtml, '\n'))
}

// scriptNonceAttrs returns a set of attributes containing a nonce attribute
// that matches the nonce specified in the Content-Security-Policy header.
//
// Script tags without their "nonce" attribute set to this value
// won't be executed by the browser.
//
// See https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP#nonces
// for details.
func scriptNonceAttrs(h http.Header) []html.Attribute {
	csp := h.Get("Content-Security-Policy")
	nonce := cspScriptNonce(csp)
	if nonce == "" {
		return nil
	}
	return []html.Attribute{
		{
			Key: "nonce",
			Val: nonce,
		},
	}
}

// cspScriptNonce parses a "Content-Security-Policy" http header value
// and extracts the script-src nonce value from it if available.
func cspScriptNonce(csp string) string {
	for _, directive := range strings.Split(csp, ";") {
		fields := strings.Fields(directive)
		if len(fields) < 2 { // This also skips empty slices, preventing panic.
			continue
		}
		if fields[0] != "script-src" {
			continue
		}
		for _, field := range fields[1:] {
			field = strings.TrimPrefix(field, "'")
			field = strings.TrimSuffix(field, "'")
			nonce, found := strings.CutPrefix(field, "nonce-")
			if found {
				return nonce
			}
		}
	}
	return ""
}

// getScript returns javascript code
// that listens to the [Server-Sent Events] emitted at eventURL
// and reloads the page if an event with type "message" and data "reload" is received.
//
// [Server-Sent Events]: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events
func getScript(eventURL string) string {
	script := `
(new EventSource("{URL}")).onmessage = function(msg) {
	if (msg && msg.data === "reload") {
		window.location.reload()
	}
}
`
	eventURL = template.JSEscapeString(eventURL)
	script = strings.ReplaceAll(script, "{URL}", eventURL)
	return script
}

// ==========

type Option func(h *Handler)

// WithDisableCaching configures whether to direct browsers
// to not cache our responses.
//
// Defaults to true.
func WithDisableCaching(v bool) Option {
	return func(h *Handler) {
		h.disableCaching = v
	}
}

// WithEventPath sets the path of the reload events webpages listen to.
// Set it to something that doesn't shadow the paths of the upstream.
//
// Defaults to "/livereloadevents".
func WithEventPath(path string) Option {
	return func(h *Handler) {
		h.eventPath = path
	}
}

// ==========

// ReverseProxy returns an [http.Handler]
// that sends it's requests to the given upstream URL
// and returns it's responses.
func ReverseProxy(upstream *url.URL) http.Handler {
	p := httputil.NewSingleHostReverseProxy(upstream)
	p.Transport = retrier.New(500*time.Millisecond, 10*time.Second)
	origDirector := p.Director
	p.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = ""
	}
	return p
}
