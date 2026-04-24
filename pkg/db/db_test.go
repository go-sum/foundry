package db

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgErr constructs a *pgconn.PgError with the given SQLSTATE code.
func pgErr(code string) *pgconn.PgError {
	return &pgconn.PgError{Code: code}
}

// --- isPgCode (tested indirectly through the exported classifiers) ---

// --- IsUniqueViolation ---

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unique violation code 23505",
			err:  pgErr("23505"),
			want: true,
		},
		{
			name: "foreign key code 23503 is not unique violation",
			err:  pgErr("23503"),
			want: false,
		},
		{
			name: "plain error is not unique violation",
			err:  errors.New("something"),
			want: false,
		},
		{
			name: "nil error is not unique violation",
			err:  nil,
			want: false,
		},
		{
			name: "wrapped unique violation unwraps correctly",
			err:  fmt.Errorf("wrap: %w", pgErr("23505")),
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsUniqueViolation(tc.err)
			if got != tc.want {
				t.Fatalf("IsUniqueViolation(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// --- IsForeignKeyViolation ---

func TestIsForeignKeyViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "foreign key code 23503",
			err:  pgErr("23503"),
			want: true,
		},
		{
			name: "unique violation code 23505 is not FK violation",
			err:  pgErr("23505"),
			want: false,
		},
		{
			name: "plain error is not FK violation",
			err:  errors.New("something"),
			want: false,
		},
		{
			name: "nil error is not FK violation",
			err:  nil,
			want: false,
		},
		{
			name: "wrapped FK violation unwraps correctly",
			err:  fmt.Errorf("wrap: %w", pgErr("23503")),
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsForeignKeyViolation(tc.err)
			if got != tc.want {
				t.Fatalf("IsForeignKeyViolation(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// --- IsDeadlock ---

func TestIsDeadlock(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "deadlock code 40P01",
			err:  pgErr("40P01"),
			want: true,
		},
		{
			name: "serialization failure 40001 is not deadlock",
			err:  pgErr("40001"),
			want: false,
		},
		{
			name: "plain error is not deadlock",
			err:  errors.New("something"),
			want: false,
		},
		{
			name: "nil error is not deadlock",
			err:  nil,
			want: false,
		},
		{
			name: "wrapped deadlock unwraps correctly",
			err:  fmt.Errorf("wrap: %w", pgErr("40P01")),
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsDeadlock(tc.err)
			if got != tc.want {
				t.Fatalf("IsDeadlock(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// --- IsSerializationFailure ---

func TestIsSerializationFailure(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "serialization failure code 40001",
			err:  pgErr("40001"),
			want: true,
		},
		{
			name: "deadlock 40P01 is not serialization failure",
			err:  pgErr("40P01"),
			want: false,
		},
		{
			name: "plain error is not serialization failure",
			err:  errors.New("something"),
			want: false,
		},
		{
			name: "nil error is not serialization failure",
			err:  nil,
			want: false,
		},
		{
			name: "wrapped serialization failure unwraps correctly",
			err:  fmt.Errorf("wrap: %w", pgErr("40001")),
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsSerializationFailure(tc.err)
			if got != tc.want {
				t.Fatalf("IsSerializationFailure(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
