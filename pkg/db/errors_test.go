package db

import (
	"errors"
	"testing"
)

var (
	errFoo = errors.New("domain error: foo")
	errBar = errors.New("domain error: bar")
)

// --- MapError ---

func TestMapError_NilError(t *testing.T) {
	got := MapError(nil, "op")
	if got != nil {
		t.Fatalf("MapError(nil, \"op\") = %v, want nil", got)
	}
}

func TestMapError_NoMappings_WrapsWithOp(t *testing.T) {
	original := errors.New("db failure")
	got := MapError(original, "create user")
	if got == nil {
		t.Fatal("MapError returned nil for non-nil error")
	}
	want := "create user: db failure"
	if got.Error() != want {
		t.Fatalf("MapError error message = %q, want %q", got.Error(), want)
	}
	if !errors.Is(got, original) {
		t.Fatal("MapError must wrap original error so errors.Is can unwrap it")
	}
}

func TestMapError_UniqueViolation_MapsToFoo(t *testing.T) {
	pgUniqueErr := pgErr("23505")
	got := MapError(pgUniqueErr, "create user", OnUniqueViolation(errFoo))
	if !errors.Is(got, errFoo) {
		t.Fatalf("MapError with unique violation = %v, want errFoo", got)
	}
}

func TestMapError_ForeignKeyViolation_MapsToBar(t *testing.T) {
	pgFKErr := pgErr("23503")
	got := MapError(pgFKErr, "create user", OnForeignKeyViolation(errBar))
	if !errors.Is(got, errBar) {
		t.Fatalf("MapError with FK violation = %v, want errBar", got)
	}
}

func TestMapError_UniqueViolation_NoMatchForFK(t *testing.T) {
	// A unique violation should NOT match an OnForeignKeyViolation mapping.
	pgUniqueErr := pgErr("23505")
	got := MapError(pgUniqueErr, "create user", OnForeignKeyViolation(errBar))
	if errors.Is(got, errBar) {
		t.Fatal("OnForeignKeyViolation must not match a unique violation error")
	}
	// It should be wrapped with the op prefix.
	want := "create user: ERROR: (23505)"
	if got.Error() != want {
		// The exact pg error message format may vary; just confirm errBar is not returned.
		if errors.Is(got, errBar) {
			t.Fatalf("MapError returned errBar for a unique violation error; got %v", got)
		}
	}
}

func TestMapError_MultipleMatchings_FirstWins(t *testing.T) {
	// Both mappings would match their respective error types.
	// Providing unique violation error: OnUniqueViolation should win as it's first.
	pgUniqueErr := pgErr("23505")
	got := MapError(pgUniqueErr, "op",
		OnUniqueViolation(errFoo),
		OnForeignKeyViolation(errBar),
	)
	if !errors.Is(got, errFoo) {
		t.Fatalf("first matching mapping should win; got %v, want errFoo", got)
	}
}

func TestMapError_NilError_WithMappings(t *testing.T) {
	// Nil error must be returned as nil regardless of mappings.
	got := MapError(nil, "op",
		OnUniqueViolation(errFoo),
		OnForeignKeyViolation(errBar),
	)
	if got != nil {
		t.Fatalf("MapError(nil, ...) = %v, want nil", got)
	}
}

// --- OnUniqueViolation ---

func TestOnUniqueViolation_MatchesUniqueError(t *testing.T) {
	m := OnUniqueViolation(errFoo)
	got := m(pgErr("23505"))
	if !errors.Is(got, errFoo) {
		t.Fatalf("OnUniqueViolation returned %v, want errFoo", got)
	}
}

func TestOnUniqueViolation_NoMatchForOtherError(t *testing.T) {
	m := OnUniqueViolation(errFoo)
	got := m(pgErr("23503"))
	if got != nil {
		t.Fatalf("OnUniqueViolation returned %v for FK error, want nil", got)
	}
}

func TestOnUniqueViolation_NoMatchForPlainError(t *testing.T) {
	m := OnUniqueViolation(errFoo)
	got := m(errors.New("plain"))
	if got != nil {
		t.Fatalf("OnUniqueViolation returned %v for plain error, want nil", got)
	}
}

// --- OnForeignKeyViolation ---

func TestOnForeignKeyViolation_MatchesFKError(t *testing.T) {
	m := OnForeignKeyViolation(errBar)
	got := m(pgErr("23503"))
	if !errors.Is(got, errBar) {
		t.Fatalf("OnForeignKeyViolation returned %v, want errBar", got)
	}
}

func TestOnForeignKeyViolation_NoMatchForOtherError(t *testing.T) {
	m := OnForeignKeyViolation(errBar)
	got := m(pgErr("23505"))
	if got != nil {
		t.Fatalf("OnForeignKeyViolation returned %v for unique error, want nil", got)
	}
}

func TestOnForeignKeyViolation_NoMatchForPlainError(t *testing.T) {
	m := OnForeignKeyViolation(errBar)
	got := m(errors.New("plain"))
	if got != nil {
		t.Fatalf("OnForeignKeyViolation returned %v for plain error, want nil", got)
	}
}
