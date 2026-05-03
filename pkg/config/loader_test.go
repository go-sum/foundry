package config

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
)

type testEnv string

const (
	testProd testEnv = "prod"
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
		Base:  baseOK,
		Rules: nil,
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
	p := testParams()
	p.Rules = nil
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
	var rulesPort int
	p.Rules = func(cfg testConfig) []func(*validator.Validate) {
		rulesPort = cfg.Port
		return nil
	}
	_, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rulesPort != 8080 {
		t.Errorf("Rules received Port=%d, want 8080", rulesPort)
	}
}
