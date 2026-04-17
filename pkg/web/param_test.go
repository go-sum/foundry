package web

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestParamInt(t *testing.T) {
	c := NewContext(context.Background(), Request{})
	c.SetParam("id", "42")
	got, err := ParamInt(c, "id")
	if err != nil {
		t.Fatalf("ParamInt() error = %v, want nil", err)
	}
	if got != 42 {
		t.Fatalf("ParamInt() = %d, want %d", got, 42)
	}
}

func TestParamInt_Error(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"missing", ""},
		{"not an integer", "abc"},
		{"float", "1.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContext(context.Background(), Request{})
			if tt.value != "" {
				c.SetParam("id", tt.value)
			}
			_, err := ParamInt(c, "id")
			if err == nil {
				t.Fatal("ParamInt() error = nil, want non-nil")
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
		})
	}
}

func TestParamUUID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
		wantVal string
	}{
		{
			name:    "v4 UUID mixed case",
			value:   "550E8400-E29B-41D4-A716-446655440000",
			wantErr: false,
			wantVal: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:    "v1 UUID",
			value:   "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			wantErr: false,
			wantVal: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		},
		{
			name:    "v7 UUID",
			value:   "018e4f08-c1c7-7000-abcd-ef0123456789",
			wantErr: false,
			wantVal: "018e4f08-c1c7-7000-abcd-ef0123456789",
		},
		{
			name:    "nil UUID",
			value:   "00000000-0000-0000-0000-000000000000",
			wantErr: false,
			wantVal: "00000000-0000-0000-0000-000000000000",
		},
		{
			name:    "missing parameter",
			value:   "",
			wantErr: true,
		},
		{
			name:    "not a UUID",
			value:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "too short",
			value:   "550e8400-e29b-41d4",
			wantErr: true,
		},
		{
			name:    "wrong dash positions",
			value:   "550e8400e29b41d4a716446655440000",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContext(context.Background(), Request{})
			if tt.value != "" {
				c.SetParam("id", tt.value)
			}
			got, err := ParamUUID(c, "id")
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParamUUID() error = nil, want non-nil")
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
					t.Fatalf("ParamUUID() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("ParamUUID() = %q, want %q", got, tt.wantVal)
				}
			}
		})
	}
}

func TestParamBool(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
		wantVal bool
	}{
		{"true string", "true", false, true},
		{"True mixed case", "True", false, true},
		{"1", "1", false, true},
		{"yes", "yes", false, true},
		{"false string", "false", false, false},
		{"False mixed case", "False", false, false},
		{"0", "0", false, false},
		{"no", "no", false, false},
		{"invalid", "maybe", true, false},
		{"missing", "", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContext(context.Background(), Request{})
			if tt.value != "" {
				c.SetParam("flag", tt.value)
			}
			got, err := ParamBool(c, "flag")
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParamBool() error = nil, want non-nil")
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
					t.Fatalf("ParamBool() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("ParamBool() = %v, want %v", got, tt.wantVal)
				}
			}
		})
	}
}

func TestParamTime(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
		wantVal time.Time
	}{
		{
			name:    "valid RFC3339",
			value:   "2026-04-16T12:00:00Z",
			wantErr: false,
			wantVal: time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC),
		},
		{
			name:    "invalid format",
			value:   "2026-04-16",
			wantErr: true,
		},
		{
			name:    "missing",
			value:   "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContext(context.Background(), Request{})
			if tt.value != "" {
				c.SetParam("ts", tt.value)
			}
			got, err := ParamTime(c, "ts")
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParamTime() error = nil, want non-nil")
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
					t.Fatalf("ParamTime() error = %v, want nil", err)
				}
				if !got.Equal(tt.wantVal) {
					t.Errorf("ParamTime() = %v, want %v", got, tt.wantVal)
				}
			}
		})
	}
}

func TestParamEnum(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		allowed []string
		wantErr bool
		wantVal string
	}{
		{
			name:    "exact match",
			value:   "asc",
			allowed: []string{"asc", "desc"},
			wantErr: false,
			wantVal: "asc",
		},
		{
			name:    "case-insensitive match",
			value:   "DESC",
			allowed: []string{"asc", "desc"},
			wantErr: false,
			wantVal: "desc",
		},
		{
			name:    "not in allowed",
			value:   "random",
			allowed: []string{"asc", "desc"},
			wantErr: true,
		},
		{
			name:    "missing parameter",
			value:   "",
			allowed: []string{"asc", "desc"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContext(context.Background(), Request{})
			if tt.value != "" {
				c.SetParam("order", tt.value)
			}
			got, err := ParamEnum(c, "order", tt.allowed...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParamEnum() error = nil, want non-nil")
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
					t.Fatalf("ParamEnum() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("ParamEnum() = %q, want %q", got, tt.wantVal)
				}
			}
		})
	}
}

func TestRequestQuery(t *testing.T) {
	u, _ := url.Parse("/search?q=hello&page=2")
	req := NewRequest(http.MethodGet, u)
	q := req.Query()
	if q.Get("q") != "hello" || q.Get("page") != "2" {
		t.Fatalf("query = %v", q)
	}
}
