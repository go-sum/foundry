// Package kv defines a storage-agnostic key-value store contract.
//
// Implementations live in sub-packages (e.g. [redisstore]). The interface
// is intentionally minimal: basic CRUD, TTL-based expiry, existence checks,
// and prefix scanning. Transactions, watches, and pub/sub are out of scope.
//
// Sentinel errors:
//   - [ErrNotFound]: returned when a requested key does not exist.
//   - [ErrClosed]: returned when an operation is attempted on a closed store.
package kv
