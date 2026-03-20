package internal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/service"
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
	}), &fakeAdminTokenValidator{})

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

func TestRegisterHTTPRoutesProtectsAdminEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	called := false
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}), &fakeAdminTokenValidator{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/issues/sync-status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if called {
		t.Fatal("gateway handler should not be called without token")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	assertAdminAuthError(t, rec, "admin_auth_missing_token", "missing bearer token")
}

func TestRegisterHTTPRoutesAllowsAdminLoginWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	called := false
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}), &fakeAdminTokenValidator{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth:login", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !called {
		t.Fatal("gateway handler should be called for login route")
	}
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

func TestRegisterHTTPRoutesResetsNoRouteStatusBeforeGatewaySuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}), &fakeAdminTokenValidator{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth:login", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != `{"success":true}` {
		t.Fatalf("body = %q, want %q", rec.Body.String(), `{"success":true}`)
	}
}

func TestRegisterHTTPRoutesAllowsAdminEndpointsWithValidBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	called := false
	store := &fakeAdminTokenValidator{user: "admin"}
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}), store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/issues/sync-status", nil)
	req.Header.Set("Authorization", "Bearer tok-123")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !called {
		t.Fatal("gateway handler should be called with valid token")
	}
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if store.lastToken != "tok-123" {
		t.Fatalf("validated token = %q, want tok-123", store.lastToken)
	}
}

func TestRegisterHTTPRoutesRejectsRevokedBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	called := false
	store := &fakeAdminTokenValidator{err: errors.New("revoked")}
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}), store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/issues/sync-status", nil)
	req.Header.Set("Authorization", "Bearer tok-123")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if called {
		t.Fatal("gateway handler should not be called with revoked token")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	assertAdminAuthError(t, rec, "admin_auth_validation_failed", "admin auth validation failed")
}

func TestRegisterHTTPRoutesHandlesCORSPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	called := false
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}), &fakeAdminTokenValidator{})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/auth:login", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if called {
		t.Fatal("gateway handler should not be called for preflight")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

func TestRegisterHTTPRoutesServesAdsTxtFromConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prev := conf.Conf.AdsTxt
	conf.Conf.AdsTxt = "example.com, pub-1234567890, DIRECT, f08c47fec0942fa0"
	t.Cleanup(func() {
		conf.Conf.AdsTxt = prev
	})

	router := gin.New()
	called := false
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}), &fakeAdminTokenValidator{})

	req := httptest.NewRequest(http.MethodGet, "/ads.txt", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if called {
		t.Fatal("gateway handler should not be called for /ads.txt")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want %q", got, "text/plain; charset=utf-8")
	}
	if rec.Body.String() != conf.Conf.AdsTxt {
		t.Fatalf("body = %q, want %q", rec.Body.String(), conf.Conf.AdsTxt)
	}
}

type fakeAdminTokenValidator struct {
	lastToken string
	user      string
	err       error
}

func assertAdminAuthError(t *testing.T, rec *httptest.ResponseRecorder, code, message string) {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if body["code"] != code {
		t.Fatalf("code = %q, want %q", body["code"], code)
	}
	if body["message"] != message {
		t.Fatalf("message = %q, want %q", body["message"], message)
	}
}

func (s *fakeAdminTokenValidator) SaveToken(context.Context, string, string, time.Duration) error {
	return nil
}

func (s *fakeAdminTokenValidator) DeleteToken(context.Context, string) error {
	return nil
}

func (s *fakeAdminTokenValidator) GetSession(context.Context, string) (service.AdminSession, error) {
	return service.AdminSession{}, nil
}

func (s *fakeAdminTokenValidator) ValidateToken(_ context.Context, token string) (string, error) {
	s.lastToken = token
	if s.err != nil {
		return "", s.err
	}
	return s.user, nil
}
