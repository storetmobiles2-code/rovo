// SPDX-License-Identifier: Apache-2.0

// Command healthcheck is a tiny static HTTP probe used as the container
// HEALTHCHECK for the distroless rovo image, which has no shell, curl or wget.
//
// Usage: healthcheck [url]   (default http://127.0.0.1:9090/healthz)
//
// It exits 0 when the URL answers 2xx within the timeout, 1 otherwise. It
// only talks to loopback by default and never follows redirects.
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	url := "http://127.0.0.1:9090/healthz"
	if v := os.Getenv("ROVO_HEALTHCHECK_URL"); v != "" {
		url = v
	}
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		os.Exit(1)
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		os.Exit(1)
	}
}
