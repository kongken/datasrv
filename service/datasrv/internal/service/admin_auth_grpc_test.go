package service

import (
	"context"
	"testing"
	"time"

	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAdminAuthGRPCServer_AdminLoginSuccess(t *testing.T) {
	store := &fakeAdminTokenStore{}
	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	srv := NewAdminAuthGRPCServer(&conf.Config{
		Admin: conf.AdminConfig{User: "admin", Password: "secret"},
	}, store)
	srv.now = func() time.Time { return now }
	srv.newToken = func() (string, error) { return "tok-123", nil }

	resp, err := srv.AdminLogin(context.Background(), &issuesv1.AdminLoginRequest{
		User:     "admin",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("AdminLogin() error = %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatal("success = false, want true")
	}
	if resp.GetToken() != "tok-123" {
		t.Fatalf("token = %q, want tok-123", resp.GetToken())
	}
	if got := resp.GetExpiresAt().AsTime(); !got.Equal(now.Add(defaultAdminTokenTTL)) {
		t.Fatalf("expires_at = %v, want %v", got, now.Add(defaultAdminTokenTTL))
	}
	if store.lastToken != "tok-123" {
		t.Fatalf("stored token = %q, want tok-123", store.lastToken)
	}
	if store.lastUser != "admin" {
		t.Fatalf("stored user = %q, want admin", store.lastUser)
	}
	if store.lastTTL != defaultAdminTokenTTL {
		t.Fatalf("stored ttl = %v, want %v", store.lastTTL, defaultAdminTokenTTL)
	}
}

func TestAdminAuthGRPCServer_AdminLoginValidationAndAuthFailures(t *testing.T) {
	srv := NewAdminAuthGRPCServer(&conf.Config{
		Admin: conf.AdminConfig{User: "admin", Password: "secret"},
	}, &fakeAdminTokenStore{})

	testCases := []struct {
		name string
		req  *issuesv1.AdminLoginRequest
		code codes.Code
	}{
		{
			name: "missing user",
			req:  &issuesv1.AdminLoginRequest{Password: "secret"},
			code: codes.InvalidArgument,
		},
		{
			name: "missing password",
			req:  &issuesv1.AdminLoginRequest{User: "admin"},
			code: codes.InvalidArgument,
		},
		{
			name: "wrong user",
			req:  &issuesv1.AdminLoginRequest{User: "root", Password: "secret"},
			code: codes.Unauthenticated,
		},
		{
			name: "wrong password",
			req:  &issuesv1.AdminLoginRequest{User: "admin", Password: "bad"},
			code: codes.Unauthenticated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := srv.AdminLogin(context.Background(), tc.req)
			if err == nil {
				t.Fatal("AdminLogin() error = nil, want error")
			}
			if status.Code(err) != tc.code {
				t.Fatalf("status code = %v, want %v", status.Code(err), tc.code)
			}
		})
	}
}

func TestAdminAuthGRPCServer_AdminLoginWithoutConfiguredCredentials(t *testing.T) {
	srv := NewAdminAuthGRPCServer(&conf.Config{}, &fakeAdminTokenStore{})

	_, err := srv.AdminLogin(context.Background(), &issuesv1.AdminLoginRequest{
		User:     "admin",
		Password: "secret",
	})
	if err == nil {
		t.Fatal("AdminLogin() error = nil, want error")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
}

func TestAdminAuthGRPCServer_AdminLoginStoreFailure(t *testing.T) {
	srv := NewAdminAuthGRPCServer(&conf.Config{
		Admin: conf.AdminConfig{User: "admin", Password: "secret"},
	}, &fakeAdminTokenStore{err: context.DeadlineExceeded})
	srv.newToken = func() (string, error) { return "tok-123", nil }

	_, err := srv.AdminLogin(context.Background(), &issuesv1.AdminLoginRequest{
		User:     "admin",
		Password: "secret",
	})
	if err == nil {
		t.Fatal("AdminLogin() error = nil, want error")
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestAdminAuthGRPCServer_AdminLogoutSuccess(t *testing.T) {
	store := &fakeAdminTokenStore{}
	srv := NewAdminAuthGRPCServer(&conf.Config{}, store)

	resp, err := srv.AdminLogout(context.Background(), &issuesv1.AdminLogoutRequest{
		Token: "tok-123",
	})
	if err != nil {
		t.Fatalf("AdminLogout() error = %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatal("success = false, want true")
	}
	if store.deletedToken != "tok-123" {
		t.Fatalf("deleted token = %q, want tok-123", store.deletedToken)
	}
}

func TestAdminAuthGRPCServer_AdminLogoutValidationAndStoreFailure(t *testing.T) {
	srv := NewAdminAuthGRPCServer(&conf.Config{}, &fakeAdminTokenStore{deleteErr: context.DeadlineExceeded})

	if _, err := srv.AdminLogout(context.Background(), &issuesv1.AdminLogoutRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing token status = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	_, err := srv.AdminLogout(context.Background(), &issuesv1.AdminLogoutRequest{Token: "tok-123"})
	if err == nil {
		t.Fatal("AdminLogout() error = nil, want error")
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestAdminAuthGRPCServer_AdminWhoAmI(t *testing.T) {
	expiresAt := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	srv := NewAdminAuthGRPCServer(&conf.Config{}, &fakeAdminTokenStore{
		sessionUser:      "admin",
		sessionExpiresAt: expiresAt,
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer tok-123"))
	resp, err := srv.AdminWhoAmI(ctx, &issuesv1.AdminWhoAmIRequest{})
	if err != nil {
		t.Fatalf("AdminWhoAmI() error = %v", err)
	}
	if resp.GetUser() != "admin" {
		t.Fatalf("user = %q, want admin", resp.GetUser())
	}
	if got := resp.GetExpiresAt().AsTime(); !got.Equal(expiresAt) {
		t.Fatalf("expires_at = %v, want %v", got, expiresAt)
	}
}

func TestAdminAuthGRPCServer_AdminWhoAmIAuthFailures(t *testing.T) {
	testCases := []struct {
		name  string
		ctx   context.Context
		store *fakeAdminTokenStore
		code  codes.Code
	}{
		{
			name:  "missing authorization metadata",
			ctx:   context.Background(),
			store: &fakeAdminTokenStore{},
			code:  codes.Unauthenticated,
		},
		{
			name: "invalid token",
			ctx:  metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer tok-123")),
			store: &fakeAdminTokenStore{
				sessionErr: ErrAdminTokenNotFound,
			},
			code: codes.Unauthenticated,
		},
		{
			name: "store failure",
			ctx:  metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer tok-123")),
			store: &fakeAdminTokenStore{
				sessionErr: context.DeadlineExceeded,
			},
			code: codes.Internal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := NewAdminAuthGRPCServer(&conf.Config{}, tc.store)
			_, err := srv.AdminWhoAmI(tc.ctx, &issuesv1.AdminWhoAmIRequest{})
			if err == nil {
				t.Fatal("AdminWhoAmI() error = nil, want error")
			}
			if status.Code(err) != tc.code {
				t.Fatalf("status code = %v, want %v", status.Code(err), tc.code)
			}
		})
	}
}

type fakeAdminTokenStore struct {
	lastToken        string
	lastUser         string
	lastTTL          time.Duration
	err              error
	deletedToken     string
	deleteErr        error
	sessionUser      string
	sessionErr       error
	sessionExpiresAt time.Time
}

func (s *fakeAdminTokenStore) SaveToken(_ context.Context, token, user string, ttl time.Duration) error {
	s.lastToken = token
	s.lastUser = user
	s.lastTTL = ttl
	return s.err
}

func (s *fakeAdminTokenStore) ValidateToken(context.Context, string) (string, error) {
	return "", nil
}

func (s *fakeAdminTokenStore) DeleteToken(_ context.Context, token string) error {
	s.deletedToken = token
	return s.deleteErr
}

func (s *fakeAdminTokenStore) GetSession(_ context.Context, token string) (AdminSession, error) {
	s.lastToken = token
	if s.sessionErr != nil {
		return AdminSession{}, s.sessionErr
	}
	return AdminSession{
		User:      s.sessionUser,
		ExpiresAt: s.sessionExpiresAt,
	}, nil
}
