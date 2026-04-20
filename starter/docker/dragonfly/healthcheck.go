// healthcheck is a minimal static binary that sends a Redis PING
// and verifies a PONG response. Port defaults to 6379 unless
// HEALTHCHECK_PORT is set. If /run/secrets/KV_PASSWORD exists, an
// AUTH command is sent first so the check works when requirepass is set.
package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	port := os.Getenv("HEALTHCHECK_PORT")
	if port == "" {
		port = "6379"
	}

	conn, err := net.DialTimeout("tcp", "localhost:"+port, 2*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	if pw, err := os.ReadFile("/run/secrets/KV_PASSWORD"); err == nil {
		password := strings.TrimSpace(string(pw))
		if password != "" {
			_, err = fmt.Fprintf(conn, "AUTH %s\r\n", password)
			if err != nil {
				fmt.Fprintf(os.Stderr, "auth write: %v\n", err)
				os.Exit(1)
			}
			buf := make([]byte, 64)
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Fprintf(os.Stderr, "auth read: %v\n", err)
				os.Exit(1)
			}
			resp := strings.TrimSpace(string(buf[:n]))
			if resp != "+OK" {
				fmt.Fprintf(os.Stderr, "auth failed: %s\n", resp)
				os.Exit(1)
			}
		}
	}

	_, err = conn.Write([]byte("PING\r\n"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}

	resp := strings.TrimSpace(string(buf[:n]))
	if resp == "+PONG" {
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "unexpected: %s\n", resp)
	os.Exit(1)
}
