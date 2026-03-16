package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	redisstore "butterfly.orx.me/core/store/redis"
	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultAdminRedisName = "default"
	defaultAdminTokenTTL  = 24 * time.Hour
	adminTokenKeyPrefix   = "datasrv:admin:token:"
)

var ErrAdminTokenNotFound = errors.New("admin token not found")

type AdminSession struct {
	User      string
	ExpiresAt time.Time
}

type AdminTokenStore interface {
	SaveToken(ctx context.Context, token, user string, ttl time.Duration) error
	ValidateToken(ctx context.Context, token string) (string, error)
	DeleteToken(ctx context.Context, token string) error
	GetSession(ctx context.Context, token string) (AdminSession, error)
}

type RedisAdminTokenStore struct {
	client *redis.Client
}

// AdminAuthGRPCServer implements issues.v1.AdminAuthService.
type AdminAuthGRPCServer struct {
	issuesv1.UnimplementedAdminAuthServiceServer

	cfg      *conf.Config
	tokens   AdminTokenStore
	now      func() time.Time
	newToken func() (string, error)
}

func NewRedisAdminTokenStore(cfg *conf.Config) *RedisAdminTokenStore {
	name := strings.TrimSpace(cfg.Admin.RedisName)
	if name == "" {
		name = defaultAdminRedisName
	}
	return &RedisAdminTokenStore{client: redisstore.GetClient(name)}
}

func NewAdminAuthGRPCServer(cfg *conf.Config, tokens AdminTokenStore) *AdminAuthGRPCServer {
	return &AdminAuthGRPCServer{
		cfg:      cfg,
		tokens:   tokens,
		now:      time.Now,
		newToken: generateAdminToken,
	}
}

func (s *AdminAuthGRPCServer) AdminLogin(ctx context.Context, req *issuesv1.AdminLoginRequest) (*issuesv1.AdminLoginResponse, error) {
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
	if s.tokens == nil {
		return nil, status.Error(codes.Internal, "admin token store is not configured")
	}

	token, err := s.newToken()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate admin token: %v", err)
	}
	now := s.now().UTC()
	ttl := adminTokenTTL(s.cfg.Admin)
	if err := s.tokens.SaveToken(ctx, token, user, ttl); err != nil {
		return nil, status.Errorf(codes.Internal, "save admin token: %v", err)
	}

	return &issuesv1.AdminLoginResponse{
		Success:   true,
		Message:   "ok",
		Token:     token,
		ExpiresAt: timestamppb.New(now.Add(ttl)),
	}, nil
}

func (s *AdminAuthGRPCServer) AdminLogout(ctx context.Context, req *issuesv1.AdminLogoutRequest) (*issuesv1.AdminLogoutResponse, error) {
	token := strings.TrimSpace(req.GetToken())
	if token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	if s.tokens == nil {
		return nil, status.Error(codes.Internal, "admin token store is not configured")
	}
	if err := s.tokens.DeleteToken(ctx, token); err != nil {
		return nil, status.Errorf(codes.Internal, "delete admin token: %v", err)
	}
	return &issuesv1.AdminLogoutResponse{
		Success: true,
		Message: "ok",
	}, nil
}

func (s *AdminAuthGRPCServer) AdminWhoAmI(ctx context.Context, _ *issuesv1.AdminWhoAmIRequest) (*issuesv1.AdminWhoAmIResponse, error) {
	if s.tokens == nil {
		return nil, status.Error(codes.Internal, "admin token store is not configured")
	}
	token := bearerTokenFromMetadata(ctx)
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "missing bearer token")
	}

	session, err := s.tokens.GetSession(ctx, token)
	if err != nil {
		if errors.Is(err, ErrAdminTokenNotFound) {
			return nil, status.Error(codes.Unauthenticated, "invalid bearer token")
		}
		return nil, status.Errorf(codes.Internal, "get admin session: %v", err)
	}

	resp := &issuesv1.AdminWhoAmIResponse{
		User: session.User,
	}
	if !session.ExpiresAt.IsZero() {
		resp.ExpiresAt = timestamppb.New(session.ExpiresAt)
	}
	return resp, nil
}

func (s *RedisAdminTokenStore) SaveToken(ctx context.Context, token, user string, ttl time.Duration) error {
	if s.client == nil {
		return errors.New("redis client is not configured")
	}
	return s.client.Set(ctx, adminTokenKeyPrefix+token, user, ttl).Err()
}

func (s *RedisAdminTokenStore) ValidateToken(ctx context.Context, token string) (string, error) {
	session, err := s.GetSession(ctx, token)
	if err != nil {
		return "", err
	}
	return session.User, nil
}

func (s *RedisAdminTokenStore) DeleteToken(ctx context.Context, token string) error {
	if s.client == nil {
		return errors.New("redis client is not configured")
	}
	return s.client.Del(ctx, adminTokenKeyPrefix+token).Err()
}

func (s *RedisAdminTokenStore) GetSession(ctx context.Context, token string) (AdminSession, error) {
	if s.client == nil {
		return AdminSession{}, errors.New("redis client is not configured")
	}

	key := adminTokenKeyPrefix + token
	user, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return AdminSession{}, ErrAdminTokenNotFound
	}
	if err != nil {
		return AdminSession{}, err
	}

	ttl, err := s.client.TTL(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return AdminSession{}, ErrAdminTokenNotFound
	}
	if err != nil {
		return AdminSession{}, err
	}

	session := AdminSession{User: user}
	if ttl > 0 {
		session.ExpiresAt = time.Now().UTC().Add(ttl)
	}
	return session, nil
}

func adminTokenTTL(cfg conf.AdminConfig) time.Duration {
	if cfg.TokenTTLSeconds <= 0 {
		return defaultAdminTokenTTL
	}
	return time.Duration(cfg.TokenTTLSeconds) * time.Second
}

func generateAdminToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func bearerTokenFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return ""
	}
	if !strings.HasPrefix(values[0], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(values[0], "Bearer "))
}
