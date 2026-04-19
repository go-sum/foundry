package secure

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-sum/web"
)

func TestClearSiteData_DefaultDirectives(t *testing.T) {
	handler := ClearSiteData()(func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusNoContent), nil
	})

	resp, err := handler(web.NewContext(context.Background(), web.Request{}))
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if got, want := resp.Headers.Get("Clear-Site-Data"), `"cookies", "storage"`; got != want {
		t.Fatalf("Clear-Site-Data = %q, want %q", got, want)
	}
}

func TestClearSiteData_CustomDirectives(t *testing.T) {
	handler := ClearSiteData("cache", "executionContexts")(func(_ *web.Context) (web.Response, error) {
		return web.Respond(http.StatusNoContent), nil
	})

	resp, err := handler(web.NewContext(context.Background(), web.Request{}))
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if got, want := resp.Headers.Get("Clear-Site-Data"), `"cache", "executionContexts"`; got != want {
		t.Fatalf("Clear-Site-Data = %q, want %q", got, want)
	}
}

func TestSetClearSiteData(t *testing.T) {
	resp := web.Respond(http.StatusOK)
	SetClearSiteData(&resp)
	if got, want := resp.Headers.Get("Clear-Site-Data"), `"cookies", "storage"`; got != want {
		t.Fatalf("default Clear-Site-Data = %q, want %q", got, want)
	}

	SetClearSiteData(&resp, "cache")
	if got, want := resp.Headers.Get("Clear-Site-Data"), `"cache"`; got != want {
		t.Fatalf("custom Clear-Site-Data = %q, want %q", got, want)
	}
}
