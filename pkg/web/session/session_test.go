package session

import (
	"encoding/json"
	"testing"
)

func TestSession_SetGet(t *testing.T) {
	s := newSession()
	if err := s.Set("name", "alice"); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	got, ok, err := Get[string](s, "name")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if !ok {
		t.Fatal("Get ok = false, want true")
	}
	if got != "alice" {
		t.Fatalf("Get = %q, want %q", got, "alice")
	}
}

func TestSession_Has(t *testing.T) {
	s := newSession()
	if s.Has("x") {
		t.Fatal("Has('x') = true before Set")
	}
	_ = s.Set("x", 1)
	if !s.Has("x") {
		t.Fatal("Has('x') = false after Set")
	}
}

func TestSession_Unset(t *testing.T) {
	s := newSession()
	_ = s.Set("k", "v")
	s.Unset("k")
	if s.Has("k") {
		t.Fatal("Has('k') = true after Unset")
	}
}

func TestSession_Keys(t *testing.T) {
	s := newSession()
	_ = s.Set("b", 2)
	_ = s.Set("a", 1)
	keys := s.Keys()
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Fatalf("Keys = %v, want [a b]", keys)
	}
}

func TestSession_Flash(t *testing.T) {
	// Simulate flash across two requests via serialization.
	s1 := newSession()
	if err := s1.Flash("msg", "hello"); err != nil {
		t.Fatalf("Flash error: %v", err)
	}

	data, err := s1.marshalPayload()
	if err != nil {
		t.Fatalf("marshalPayload: %v", err)
	}

	s2 := sessionFromData(data, "tok", 0)
	msg, ok, err := FlashPop[string](s2, "msg")
	if err != nil {
		t.Fatalf("FlashPop error: %v", err)
	}
	if !ok || msg != "hello" {
		t.Fatalf("FlashPop = %q, %v, want 'hello', true", msg, ok)
	}

	// After FlashPop, key is gone.
	_, ok2, _ := FlashPop[string](s2, "msg")
	if ok2 {
		t.Fatal("FlashPop second call: ok = true, want false")
	}
}

func TestSession_FlashUnconsumedDropped(t *testing.T) {
	// Flash not consumed this request should be absent in next request.
	s1 := newSession()
	_ = s1.Flash("notice", "temporary")

	data, _ := s1.marshalPayload()
	s2 := sessionFromData(data, "tok", 0) // s2.currentFlash = {"notice": "temporary"}

	// Do not call FlashPop — unconsumed.
	data2, _ := s2.marshalPayload() // nextFlash is empty; currentFlash is dropped

	s3 := sessionFromData(data2, "tok", 0)
	_, ok, _ := FlashPop[string](s3, "notice")
	if ok {
		t.Fatal("unconsumed flash persisted across 2 requests, want dropped")
	}
}

func TestSession_Destroy(t *testing.T) {
	s := newSession()
	_ = s.Set("k", "v")
	s.Destroy()
	if s.Has("k") {
		t.Fatal("Has after Destroy = true, want false")
	}
	if !s.destroyed {
		t.Fatal("destroyed flag = false after Destroy")
	}
}

func TestSession_Regenerate(t *testing.T) {
	s := &Session{
		token:        "old-token",
		values:       make(map[string]json.RawMessage),
		currentFlash: make(map[string]json.RawMessage),
	}
	_ = s.Set("user", "alice")
	s.Regenerate()

	if s.token != "" {
		t.Fatalf("token = %q after Regenerate, want empty", s.token)
	}
	if s.oldToken != "old-token" {
		t.Fatalf("oldToken = %q, want 'old-token'", s.oldToken)
	}
	if !s.regenerated {
		t.Fatal("regenerated = false after Regenerate")
	}
	// Data preserved.
	got, ok, _ := Get[string](s, "user")
	if !ok || got != "alice" {
		t.Fatal("session data lost after Regenerate")
	}
}

func TestSession_NilSafety(t *testing.T) {
	var s *Session
	if s.ID() != "" {
		t.Fatal("nil.ID() non-empty")
	}
	if s.IsNew() {
		t.Fatal("nil.IsNew() = true")
	}
	if s.Has("x") {
		t.Fatal("nil.Has() = true")
	}
	s.Unset("x")
	s.Destroy()
	s.Regenerate()
	_ = s.Keys()

	_, _, err := Get[string](s, "x")
	if err != nil {
		t.Fatalf("nil Get error: %v", err)
	}
	_, _, err = FlashPop[string](s, "x")
	if err != nil {
		t.Fatalf("nil FlashPop error: %v", err)
	}
}

func TestSessionFromData_Corrupted(t *testing.T) {
	s := sessionFromData([]byte("not json"), "tok", 0)
	if !s.fresh {
		t.Fatal("corrupted data: session not fresh")
	}
}
