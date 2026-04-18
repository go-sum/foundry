package web

import (
	"net/http"
	"testing"
)

func TestParseScalar_String(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantVal string
		wantErr bool
	}{
		{"non-empty string", "hello", "hello", false},
		{"empty string", "", "", false}, // string type accepts empty
		{"numeric string", "42", "42", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScalar[string](tt.raw, "query parameter", "q")
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseScalar[string]() error = nil, want non-nil")
				}
			} else {
				if err != nil {
					t.Fatalf("parseScalar[string]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("parseScalar[string]() = %q, want %q", got, tt.wantVal)
				}
			}
		})
	}
}

func TestParseScalar_Int(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantVal int
		wantErr bool
	}{
		{"positive integer", "42", 42, false},
		{"zero", "0", 0, false},
		{"negative integer", "-10", -10, false},
		{"not an integer", "abc", 0, true},
		{"float", "1.5", 0, true},
		{"empty", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScalar[int](tt.raw, "query parameter", "page")
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseScalar[int]() error = nil, want non-nil")
				}
				e, ok := err.(*Error)
				if !ok {
					t.Fatalf("err type = %T, want *Error", err)
				}
				if e.Status != http.StatusBadRequest {
					t.Errorf("Status = %d, want 400", e.Status)
				}
				if e.Code != CodeBadRequest {
					t.Errorf("Code = %q, want %q", e.Code, CodeBadRequest)
				}
			} else {
				if err != nil {
					t.Fatalf("parseScalar[int]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("parseScalar[int]() = %d, want %d", got, tt.wantVal)
				}
			}
		})
	}
}

func TestParseScalar_Int64(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantVal int64
		wantErr bool
	}{
		{"positive", "9223372036854775807", 9223372036854775807, false},
		{"negative", "-1", -1, false},
		{"zero", "0", 0, false},
		{"not an integer", "xyz", 0, true},
		{"float", "3.14", 0, true},
		{"empty", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScalar[int64](tt.raw, "route parameter", "id")
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseScalar[int64]() error = nil, want non-nil")
				}
				e, ok := err.(*Error)
				if !ok {
					t.Fatalf("err type = %T, want *Error", err)
				}
				if e.Status != http.StatusBadRequest {
					t.Errorf("Status = %d, want 400", e.Status)
				}
			} else {
				if err != nil {
					t.Fatalf("parseScalar[int64]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("parseScalar[int64]() = %d, want %d", got, tt.wantVal)
				}
			}
		})
	}
}

func TestParseScalar_Float64(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantVal float64
		wantErr bool
	}{
		{"integer-like", "3", 3.0, false},
		{"decimal", "3.14", 3.14, false},
		{"negative", "-0.5", -0.5, false},
		{"not a number", "abc", 0, true},
		{"empty", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScalar[float64](tt.raw, "query parameter", "amount")
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseScalar[float64]() error = nil, want non-nil")
				}
				e, ok := err.(*Error)
				if !ok {
					t.Fatalf("err type = %T, want *Error", err)
				}
				if e.Status != http.StatusBadRequest {
					t.Errorf("Status = %d, want 400", e.Status)
				}
			} else {
				if err != nil {
					t.Fatalf("parseScalar[float64]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("parseScalar[float64]() = %v, want %v", got, tt.wantVal)
				}
			}
		})
	}
}

func TestParseScalar_Bool(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantVal bool
		wantErr bool
	}{
		{"true", "true", true, false},
		{"True mixed case", "True", true, false},
		{"1", "1", true, false},
		{"yes", "yes", true, false},
		{"false", "false", false, false},
		{"False mixed case", "False", false, false},
		{"0", "0", false, false},
		{"no", "no", false, false},
		{"invalid value", "maybe", false, true},
		{"empty", "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScalar[bool](tt.raw, "query parameter", "active")
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseScalar[bool]() error = nil, want non-nil")
				}
				e, ok := err.(*Error)
				if !ok {
					t.Fatalf("err type = %T, want *Error", err)
				}
				if e.Status != http.StatusBadRequest {
					t.Errorf("Status = %d, want 400", e.Status)
				}
			} else {
				if err != nil {
					t.Fatalf("parseScalar[bool]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("parseScalar[bool]() = %v, want %v", got, tt.wantVal)
				}
			}
		})
	}
}

func TestParseBoolLenient(t *testing.T) {
	tests := []struct {
		raw     string
		want    bool
		wantErr bool
	}{
		{"true", true, false},
		{"TRUE", true, false},
		{"1", true, false},
		{"yes", true, false},
		{"YES", true, false},
		{"false", false, false},
		{"FALSE", false, false},
		{"0", false, false},
		{"no", false, false},
		{"NO", false, false},
		{"maybe", false, true},
		{"", false, true},
		{"on", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := parseBoolLenient(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseBoolLenient() error = nil, want non-nil")
				}
			} else {
				if err != nil {
					t.Fatalf("parseBoolLenient() error = %v, want nil", err)
				}
				if got != tt.want {
					t.Errorf("parseBoolLenient() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
