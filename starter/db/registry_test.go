package db

import (
	"strings"
	"testing"
)

// TestExternalSchemas_ContainsBaseAndQueue verifies that ExternalSchemas returns
// a resolver with both the "base" and "queue" keys populated.
func TestExternalSchemas_ContainsBaseAndQueue(t *testing.T) {
	resolver := ExternalSchemas()

	if _, ok := resolver["base"]; !ok {
		t.Fatal(`ExternalSchemas() missing key "base"`)
	}
	if _, ok := resolver["queue"]; !ok {
		t.Fatal(`ExternalSchemas() missing key "queue"`)
	}
}

// TestExternalSchemas_BaseSQL_NonEmpty verifies that the SQL for the "base"
// entry is not the empty string (i.e. the embed succeeded).
func TestExternalSchemas_BaseSQL_NonEmpty(t *testing.T) {
	resolver := ExternalSchemas()

	if resolver["base"] == "" {
		t.Fatal(`ExternalSchemas()["base"] is empty; embed likely failed`)
	}
}

// TestExternalSchemas_QueueSQL_ContainsQueueJobs verifies that the SQL for the
// "queue" entry references the queue_jobs table, confirming the correct SQL
// file was embedded.
func TestExternalSchemas_QueueSQL_ContainsQueueJobs(t *testing.T) {
	resolver := ExternalSchemas()

	if !strings.Contains(resolver["queue"], "queue_jobs") {
		t.Fatalf(`ExternalSchemas()["queue"] does not contain "queue_jobs"; got: %q`, resolver["queue"])
	}
}
