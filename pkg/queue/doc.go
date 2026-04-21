// Package queue defines a background job queue with priority-based scheduling,
// pluggable persistence, and configurable retry logic.
//
// Enqueuing is done via [Dispatcher]; processing is done via [Processor].
// Store implementations live in sub-packages (e.g. pgstore).
//
// Sync mode: construct a [Dispatcher] with a nil Store to execute handlers
// inline during Dispatch — useful in tests and single-process deployments.
//
// Sentinel errors:
//   - [ErrNotFound]: no job available for dequeue.
//   - [ErrQueueUnknown]: dispatch to an unregistered queue name.
//   - [ErrClosed]: operation on a stopped Dispatcher or Processor.
//
// Out of scope: scheduled recurring jobs (cron), distributed tracing
// propagation, dead-letter queue UI.
package queue
