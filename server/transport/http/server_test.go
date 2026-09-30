package http

import (
	stdhttp "net/http"
	"testing"
)

func TestNewUsesDefaultsAndNativeHandler(t *testing.T) {
	t.Parallel()
	handler := stdhttp.NewServeMux()
	server := New("", handler)
	if server.Address() != DefaultAddress {
		t.Fatalf("Address() = %q", server.Address())
	}
	if server.HTTP().Handler != handler {
		t.Fatal("HTTP handler was not preserved")
	}
}
