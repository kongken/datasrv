package internal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func TestNewGatewayMuxReturnsRegistrarError(t *testing.T) {
	wantErr := errors.New("register failed")

	_, err := newGatewayMux(context.Background(), "localhost:9090", nil, func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("newGatewayMux() error = %v, want %v", err, wantErr)
	}
}

func TestRegisterHTTPRoutesForwardsToGatewayHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	called := false
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/api/v1/issues" {
			t.Fatalf("path = %q, want /api/v1/issues", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !called {
		t.Fatal("gateway handler was not called")
	}
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
}
