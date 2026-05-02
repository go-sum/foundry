package config

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
)

type testEnv string

const (
	testProd testEnv = "prod"
	testDev  testEnv = "dev"
)

type testConfig struct {
	Name string `validate:"required"`
	Env  testEnv
	Port int
}

func baseOK() (testConfig, error) {
	return testConfig{Name: "base", Port: 8080}, nil
}

var errBase = errors.New("base failed")

func baseErr() (testConfig, error) {
	return testConfig{}, errBase
}

func baseEmpty() (testConfig, error) {
	return testConfig{}, nil
}

func testParams() LoadParams[testConfig] {
	return LoadParams[testConfig]{
		Base: baseOK,
		Env:  string(testProd),
		Overlays: []EnvOverlay[testConfig]{
			{string(testDev), func(c *testConfig) { c.Port = 3000 }},
		},
		SetEnv: func(c *testConfig, e string) { c.Env = testEnv(e) },
	}
}

func TestLoad_BaseConfigReturned(t *testing.T) {
	cfg, err := Load(testParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "base" || cfg.Port != 8080 {
		t.Errorf("got Name=%q Port=%d, want Name=%q Port=%d", cfg.Name, cfg.Port, "base", 8080)
	}
}

func TestLoad_OverlayApplied(t *testing.T) {
	p := testParams()
	p.Env = string(testDev)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 3000 {
		t.Errorf("got Port=%d, want 3000", cfg.Port)
	}
}

func TestLoad_UnknownEnvNoOverlay(t *testing.T) {
	p := testParams()
	p.Env = "staging"
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("got Port=%d, want 8080 (no overlay for staging)", cfg.Port)
	}
}

func TestLoad_SetEnvCalled(t *testing.T) {
	p := testParams()
	p.Env = string(testDev)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Env != testDev {
		t.Errorf("got Env=%q, want %q", cfg.Env, testDev)
	}
}

func TestLoad_SetEnvNil(t *testing.T) {
	p := testParams()
	p.SetEnv = nil
	_, err := Load(p)
	if err != nil {
		t.Fatalf("nil SetEnv panicked or errored: %v", err)
	}
}

func TestLoad_BaseErrorPropagated(t *testing.T) {
	p := testParams()
	p.Base = baseErr
	_, err := Load(p)
	if !errors.Is(err, errBase) {
		t.Errorf("got %v, want errBase", err)
	}
}

func TestLoad_ValidationFails(t *testing.T) {
	p := testParams()
	p.Base = baseEmpty
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestLoad_ValidationPasses(t *testing.T) {
	cfg, err := Load(testParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil pointer")
	}
}

func TestLoad_RulesNil(t *testing.T) {
	p := testParams()
	p.Rules = nil
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "base" {
		t.Errorf("got Name=%q, want %q", cfg.Name, "base")
	}
}

func TestLoad_RulesReceiveOverlaidConfig(t *testing.T) {
	p := testParams()
	p.Env = string(testDev)
	var rulesPort int
	p.Rules = func(cfg testConfig) []func(*validator.Validate) {
		rulesPort = cfg.Port
		return nil
	}
	_, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rulesPort != 3000 {
		t.Errorf("Rules received Port=%d, want 3000 (after overlay)", rulesPort)
	}
}

func TestLoadEnv_SkipsValidation(t *testing.T) {
	p := testParams()
	p.Base = baseEmpty
	cfg, err := LoadEnv(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "" {
		t.Errorf("got Name=%q, want empty (no validation should fire)", cfg.Name)
	}
}

func TestLoadEnv_OverlayApplied(t *testing.T) {
	p := testParams()
	p.Env = string(testDev)
	cfg, err := LoadEnv(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 3000 {
		t.Errorf("got Port=%d, want 3000", cfg.Port)
	}
}

func TestLoadEnv_BaseErrorPropagated(t *testing.T) {
	p := testParams()
	p.Base = baseErr
	_, err := LoadEnv(p)
	if !errors.Is(err, errBase) {
		t.Errorf("got %v, want errBase", err)
	}
}
