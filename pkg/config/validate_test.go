package config_test

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/go-sum/foundry/pkg/config"
)

func TestValidate_RequiredField_Fails(t *testing.T) {
	type S struct {
		SomeField string `validate:"required"`
	}

	err := config.Validate(S{})
	if err == nil {
		t.Fatal("expected error for missing required field, got nil")
	}
	if !strings.Contains(err.Error(), "SomeField") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "SomeField")
	}
}

func TestValidate_MinLength_Fails(t *testing.T) {
	type S struct {
		KeyField []byte `validate:"required,min=32"`
	}

	err := config.Validate(S{KeyField: make([]byte, 30)})
	if err == nil {
		t.Fatal("expected error for min=32 with 30-byte slice, got nil")
	}
	if !strings.Contains(err.Error(), "KeyField") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "KeyField")
	}
}

func TestValidate_Valid(t *testing.T) {
	type S struct {
		SomeField string `validate:"required"`
		KeyField  []byte `validate:"required,min=32"`
	}

	err := config.Validate(S{
		SomeField: "hello",
		KeyField:  make([]byte, 32),
	})
	if err != nil {
		t.Errorf("expected nil error for valid struct, got %v", err)
	}
}

type ruleTarget struct {
	Name string
}

func TestValidate_InvokesRegistrars(t *testing.T) {
	registrar := func(v *validator.Validate) {
		v.RegisterStructValidation(func(sl validator.StructLevel) {
			sl.ReportError(sl.Current().Interface().(ruleTarget).Name, "Name", "Name", "customrule", "")
		}, ruleTarget{})
	}

	err := config.Validate(ruleTarget{Name: "anything"}, registrar)
	if err == nil {
		t.Fatal("expected error from registered rule, got nil")
	}
	if !strings.Contains(err.Error(), "Name") {
		t.Errorf("error = %q, want it to contain Name", err.Error())
	}
	if !strings.Contains(err.Error(), "customrule") {
		t.Errorf("error = %q, want it to contain customrule", err.Error())
	}
}

func TestValidate_FormatsFieldNamespace(t *testing.T) {
	type Inner struct {
		Field string `validate:"required"`
	}
	type Outer struct {
		Inner Inner
	}

	err := config.Validate(Outer{})
	if err == nil {
		t.Fatal("expected error for nested required field, got nil")
	}
	if !strings.Contains(err.Error(), "Outer.Inner.Field") {
		t.Errorf("error = %q, want it to contain full namespace Outer.Inner.Field", err.Error())
	}
}

func TestValidate_RequiredTag_HumanPhrase(t *testing.T) {
	type S struct {
		SomeField string `validate:"required"`
	}

	err := config.Validate(S{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "is required") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "is required")
	}
}

func TestValidate_MinTag_HumanPhrase(t *testing.T) {
	type S struct {
		KeyField []byte `validate:"required,min=32"`
	}

	err := config.Validate(S{KeyField: make([]byte, 30)})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "at least 32") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "at least 32")
	}
}

func TestValidate_ReadsHelpTag(t *testing.T) {
	type S struct {
		KeyField string `validate:"required" help:"set FOO_ENV in your .env file"`
	}

	err := config.Validate(S{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "help: set FOO_ENV in your .env file") {
		t.Errorf("error = %q, want it to contain help text", err.Error())
	}
}

func TestValidate_UnknownTag_FallsBackToQuoted(t *testing.T) {
	registrar := func(v *validator.Validate) {
		v.RegisterStructValidation(func(sl validator.StructLevel) {
			sl.ReportError(sl.Current().Interface().(ruleTarget).Name, "Name", "Name", "customrule", "")
		}, ruleTarget{})
	}

	err := config.Validate(ruleTarget{Name: "anything"}, registrar)
	if err == nil {
		t.Fatal("expected error from registered rule, got nil")
	}
	if !strings.Contains(err.Error(), `"customrule"`) {
		t.Errorf("error = %q, want it to contain quoted tag %q", err.Error(), `"customrule"`)
	}
}

func TestValidate_MultipleErrors_PreambleOnce(t *testing.T) {
	type S struct {
		A string `validate:"required"`
		B string `validate:"required"`
	}

	err := config.Validate(S{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	preamble := "configuration validation failed:"
	count := strings.Count(err.Error(), preamble)
	if count != 1 {
		t.Errorf("preamble appears %d times, want exactly 1; error = %q", count, err.Error())
	}
}

type ruleA struct{ X string }
type ruleB struct{ Y string }

func TestValidate_MultipleRegistrars_AllInvoked(t *testing.T) {
	type combined struct {
		A ruleA
		B ruleB
	}

	regA := func(v *validator.Validate) {
		v.RegisterStructValidation(func(sl validator.StructLevel) {
			sl.ReportError(sl.Current().Interface().(ruleA).X, "X", "X", "rulea", "")
		}, ruleA{})
	}
	regB := func(v *validator.Validate) {
		v.RegisterStructValidation(func(sl validator.StructLevel) {
			sl.ReportError(sl.Current().Interface().(ruleB).Y, "Y", "Y", "ruleb", "")
		}, ruleB{})
	}

	err := config.Validate(combined{}, regA, regB)
	if err == nil {
		t.Fatal("expected error from both registrars, got nil")
	}
	if !strings.Contains(err.Error(), "rulea") {
		t.Errorf("error = %q, want rulea", err.Error())
	}
	if !strings.Contains(err.Error(), "ruleb") {
		t.Errorf("error = %q, want ruleb", err.Error())
	}
}
