// healthcheck is a minimal static binary that sends an HTTP GET to the
// local app server's /health endpoint and exits 0 on a 2xx/3xx response.
// Port defaults to 8080 unless HEALTHCHECK_PORT is set.
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("HEALTHCHECK_PORT")
	if port == "" {
		port = "8080"
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
		},
	}

	resp, err := client.Get("http://localhost:" + port + "/health")
	if err != nil {
		fmt.Fprintf(os.Stderr, "health: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "health: status %d\n", resp.StatusCode)
	os.Exit(1)
}
