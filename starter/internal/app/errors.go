package app

import "errors"

var (
	ErrKVStoreUnavailable        = errors.New("app: kv store unavailable")
	ErrKVSessionStoreUnsupported = errors.New("app: kv session store: KV store does not implement session.KVStore")
	ErrTrustedProxyCIDRInvalid   = errors.New("app: invalid trusted proxy CIDR")
)
