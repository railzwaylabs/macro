package macro_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/railzwaylabs/macro"
)

func TestWithGatewayEnablesOfficialServeMux(t *testing.T) {
	app := macro.NewService("catalogue", macro.WithGateway())
	if app.Gateway() == nil || app.HTTP() == nil {
		t.Fatal("gateway and HTTP server must be enabled")
	}

	if err := app.Gateway().HandlePath(http.MethodGet, "/v1/health", func(http.ResponseWriter, *http.Request, map[string]string) {
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	response := httptest.NewRecorder()
	app.Gateway().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
}
