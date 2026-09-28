package logger_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	macrologger "github.com/railzwaylabs/macro/logger"
)

func TestHandlerChangesMode(t *testing.T) {
	t.Parallel()

	log, err := macrologger.New(macrologger.Config{Mode: "info"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if log.Zap().Core().Enabled(zap.DebugLevel) {
		t.Fatal("debug logging unexpectedly enabled")
	}

	request := httptest.NewRequest(http.MethodPut, "/log/mode", strings.NewReader(`{"mode":"debug"}`))
	response := httptest.NewRecorder()
	log.Handler().ServeHTTP(response, request)

	if got, want := response.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d; body = %s", got, want, response.Body.String())
	}
	if got, want := log.Mode(), "debug"; got != want {
		t.Fatalf("Mode() = %q, want %q", got, want)
	}
	if !log.Zap().Core().Enabled(zap.DebugLevel) {
		t.Fatal("debug logging was not enabled")
	}
}

func TestHandlerGetsMode(t *testing.T) {
	t.Parallel()

	log, err := macrologger.New(macrologger.Config{Mode: "warn"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/log/mode", nil)
	response := httptest.NewRecorder()
	log.Handler().ServeHTTP(response, request)

	if got, want := response.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if got, want := response.Body.String(), "{\"mode\":\"warn\"}\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestHandlerRejectsUnsupportedMode(t *testing.T) {
	t.Parallel()

	log, err := macrologger.New(macrologger.Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPut, "/log/mode", strings.NewReader(`{"mode":"fatal"}`))
	response := httptest.NewRecorder()
	log.Handler().ServeHTTP(response, request)

	if got, want := response.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if got, want := log.Mode(), "info"; got != want {
		t.Fatalf("Mode() = %q, want %q", got, want)
	}
}

func TestRegisterHandler(t *testing.T) {
	t.Parallel()

	log, err := macrologger.New(macrologger.Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	mux := http.NewServeMux()
	log.RegisterHandler(mux)

	request := httptest.NewRequest(http.MethodGet, "/log/mode", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if got, want := response.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
}
