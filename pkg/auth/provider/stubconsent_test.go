package provider_test

import (
	"testing"

	"github.com/go-sum/foundry/pkg/auth/provider"
)

func TestStubConsentRenderer_ConsentPage_ReturnsError(t *testing.T) {
	renderer := provider.NewStubConsentRenderer()
	_, err := renderer.ConsentPage(nil, provider.ConsentPageData{})
	if err == nil {
		t.Error("ConsentPage returned nil error, want non-nil")
	}
}
