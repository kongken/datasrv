package service

import (
	"context"
	"strings"

	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AdminAuthGRPCServer implements issues.v1.AdminAuthService.
type AdminAuthGRPCServer struct {
	issuesv1.UnimplementedAdminAuthServiceServer

	cfg *conf.Config
}

func NewAdminAuthGRPCServer(cfg *conf.Config) *AdminAuthGRPCServer {
	return &AdminAuthGRPCServer{cfg: cfg}
}

func (s *AdminAuthGRPCServer) AdminLogin(_ context.Context, req *issuesv1.AdminLoginRequest) (*issuesv1.AdminLoginResponse, error) {
	user := strings.TrimSpace(req.GetUser())
	password := req.GetPassword()
	if user == "" {
		return nil, status.Error(codes.InvalidArgument, "user is required")
	}
	if password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	expectedUser := strings.TrimSpace(s.cfg.Admin.User)
	expectedPassword := s.cfg.Admin.Password
	if expectedUser == "" || expectedPassword == "" {
		return nil, status.Error(codes.Unauthenticated, "admin credentials are not configured")
	}
	if user != expectedUser || password != expectedPassword {
		return nil, status.Error(codes.Unauthenticated, "invalid admin credentials")
	}

	return &issuesv1.AdminLoginResponse{
		Success: true,
		Message: "ok",
	}, nil
}
