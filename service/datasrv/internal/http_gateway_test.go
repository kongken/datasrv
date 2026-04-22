package internal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
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

func TestRegisterHTTPRoutesServesIssueStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevStore := syncStore
	now := time.Now().UTC().Truncate(time.Second)
	syncStore = &stubIssueStatsStore{
		rows: []dao.SyncedIssue{
			{Repo: "o/r1", State: "open", Comments: 2, AISummary: "summary", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-time.Hour)},
			{Repo: "o/r2", State: "closed", Comments: 5, CreatedAt: now.Add(-90 * time.Minute), UpdatedAt: now},
		},
	}
	t.Cleanup(func() {
		syncStore = prevStore
	})

	router := gin.New()
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("gateway should not be called for %s", r.URL.Path)
	}), &fakeAdminTokenValidator{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp issueStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("total = %d, want 2", resp.Total)
	}
	if resp.Open != 1 {
		t.Fatalf("open = %d, want 1", resp.Open)
	}
	if resp.Closed != 1 {
		t.Fatalf("closed = %d, want 1", resp.Closed)
	}
	if resp.WithAISummary != 1 {
		t.Fatalf("withAiSummary = %d, want 1", resp.WithAISummary)
	}
	if resp.TotalComments != 7 {
		t.Fatalf("totalComments = %d, want 7", resp.TotalComments)
	}
	if resp.RepoCount != 2 {
		t.Fatalf("repoCount = %d, want 2", resp.RepoCount)
	}
	if resp.LatestCreatedAt != now.Add(-90*time.Minute).Format(time.RFC3339Nano) {
		t.Fatalf("latestCreatedAt = %q", resp.LatestCreatedAt)
	}
	if resp.LatestUpdatedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("latestUpdatedAt = %q", resp.LatestUpdatedAt)
	}
}

func TestRegisterHTTPRoutesServesIssueStatsByRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevStore := syncStore
	syncStore = &stubIssueStatsStore{
		rows: []dao.SyncedIssue{
			{Repo: "o/r2", State: "closed", Comments: 5},
		},
	}
	t.Cleanup(func() {
		syncStore = prevStore
	})

	router := gin.New()
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("gateway should not be called for %s", r.URL.Path)
	}), &fakeAdminTokenValidator{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/stats?repo=o/r2", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := syncStore.(*stubIssueStatsStore).lastFilter.Repo; got != "o/r2" {
		t.Fatalf("repo filter = %q, want o/r2", got)
	}
}

func TestRegisterHTTPRoutesIssueStatsHandlesStoreError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevStore := syncStore
	syncStore = &stubIssueStatsStore{err: errors.New("db unavailable")}
	t.Cleanup(func() {
		syncStore = prevStore
	})

	router := gin.New()
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("gateway should not be called for %s", r.URL.Path)
	}), &fakeAdminTokenValidator{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rec.Body.String(), "issue_stats_query_failed") {
		t.Fatalf("body = %q, want query failure code", rec.Body.String())
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

func TestRegisterHTTPRoutesServesSitemapXMLFromConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prev := conf.Conf.SitemapXML
	conf.Conf.SitemapXML = `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`
	t.Cleanup(func() {
		conf.Conf.SitemapXML = prev
	})

	router := gin.New()
	called := false
	registerHTTPRoutes(router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}), &fakeAdminTokenValidator{})

	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if called {
		t.Fatal("gateway handler should not be called for /sitemap.xml")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/xml; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/xml; charset=utf-8")
	}
	if rec.Body.String() != conf.Conf.SitemapXML {
		t.Fatalf("body = %q, want %q", rec.Body.String(), conf.Conf.SitemapXML)
	}
}

type fakeAdminTokenValidator struct {
	lastToken string
	user      string
	err       error
}

type stubIssueStatsStore struct {
	rows       []dao.SyncedIssue
	err        error
	lastFilter dao.SyncIssueFilter
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

func (s *stubIssueStatsStore) UpsertIssues(context.Context, string, []dao.SyncedIssue) (int, error) {
	return 0, nil
}

func (s *stubIssueStatsStore) ListIssues(_ context.Context, filter dao.SyncIssueFilter) ([]dao.SyncedIssue, error) {
	s.lastFilter = filter
	if s.err != nil {
		return nil, s.err
	}
	return append([]dao.SyncedIssue(nil), s.rows...), nil
}

func (s *stubIssueStatsStore) UpdateIssueAISummary(context.Context, string, int64, int32, string) (dao.SyncedIssue, error) {
	return dao.SyncedIssue{}, nil
}

func (s *stubIssueStatsStore) ClearIssueAISummaries(context.Context, string) (int, error) {
	return 0, nil
}

func (s *stubIssueStatsStore) ListManagedRepos(context.Context) ([]dao.ManagedRepo, error) {
	return nil, nil
}

func (s *stubIssueStatsStore) ReplaceManagedRepos(context.Context, []string) ([]dao.ManagedRepo, error) {
	return nil, nil
}

func (s *stubIssueStatsStore) GetRepoCheckpoint(context.Context, string) (dao.Checkpoint, error) {
	return dao.Checkpoint{}, nil
}

func (s *stubIssueStatsStore) SaveRepoCheckpoint(context.Context, dao.Checkpoint) error {
	return nil
}

func (s *stubIssueStatsStore) ListCheckpoints(context.Context) ([]dao.Checkpoint, error) {
	return nil, nil
}

func (s *stubIssueStatsStore) Close() error {
	return nil
}
