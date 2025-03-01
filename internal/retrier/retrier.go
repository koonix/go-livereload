// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

// Package retrier provides an [http.RoundTripper] that retries requests.
package retrier

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Transport is an [http.RoundTripper]
// that retries the request several times until it succeeds.
type Transport struct {
	retryDelay    time.Duration
	maxRetryCount int
}

// New creates a new [Transport].
func New(retryDelay, maxRetryTime time.Duration) *Transport {
	return &Transport{
		retryDelay:    retryDelay,
		maxRetryCount: int(maxRetryTime / retryDelay),
	}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {

	// br represents the request body as a [*bytes.Reader] which is seekable.
	var br *bytes.Reader

	// Read req.Body into br.
	if req.Body != nil && req.Body != http.NoBody {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read request body: %w", err)
		}
		br = bytes.NewReader(b)
	}

	origReq := req
	var roundtripErr error

	for range t.maxRetryCount {

		// Clone the request.
		req := origReq.Clone(origReq.Context())

		// Renew the request body.
		if br != nil {
			br.Seek(0, io.SeekStart)
			req.Body = io.NopCloser(br)
		}

		// Make the request and get a response.
		resp, err := http.DefaultTransport.RoundTrip(req)

		// Retry if request failed.
		if err != nil {
			roundtripErr = err
			time.Sleep(t.retryDelay)
			continue
		}

		return resp, nil
	}

	return nil, roundtripErr
}
