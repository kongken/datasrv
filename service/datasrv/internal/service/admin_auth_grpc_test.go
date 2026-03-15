package service

import (
	"context"
	"testing"

	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAdminAuthGRPCServer_AdminLoginSuccess(t *testing.T) {
	srv := NewAdminAuthGRPCServer(&conf.Config{
		Admin: conf.AdminConfig{User: "admin", Password: "secret"},
	})

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
}

func TestAdminAuthGRPCServer_AdminLoginValidationAndAuthFailures(t *testing.T) {
	srv := NewAdminAuthGRPCServer(&conf.Config{
		Admin: conf.AdminConfig{User: "admin", Password: "secret"},
	})

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
	srv := NewAdminAuthGRPCServer(&conf.Config{})

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
