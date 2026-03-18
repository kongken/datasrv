package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	feedsv1 "github.com/kongken/datasrv/pkg/proto/feeds/v1"
	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const grpcGatewayEndpoint = "localhost:9090"

type gatewayRegistrar func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

var gatewayHandler http.Handler
var adminTokenValidator service.AdminTokenStore

func initGatewayHandler() error {
	handler, err := newGatewayMux(context.Background(), grpcGatewayEndpoint, []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}, issuesv1.RegisterIssueSyncAdminServiceHandlerFromEndpoint,
		issuesv1.RegisterIssueQueryServiceHandlerFromEndpoint,
		issuesv1.RegisterAdminAuthServiceHandlerFromEndpoint,
		feedsv1.RegisterFeedSyncAdminServiceHandlerFromEndpoint,
		feedsv1.RegisterFeedQueryServiceHandlerFromEndpoint,
	)
	if err != nil {
		return fmt.Errorf("init grpc gateway: %w", err)
	}
	gatewayHandler = handler
	return nil
}

func newGatewayMux(ctx context.Context, endpoint string, opts []grpc.DialOption, registrars ...gatewayRegistrar) (http.Handler, error) {
	mux := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(gatewayHeaderMatcher))
	for _, register := range registrars {
		if err := register(ctx, mux, endpoint, opts); err != nil {
			return nil, err
		}
	}
	return mux, nil
}

func registerHTTPRoutes(r *gin.Engine, gateway http.Handler, tokens service.AdminTokenStore) {
	r.Use(corsMiddleware())

	if gateway == nil {
		r.NoRoute(func(c *gin.Context) {
			writeAdminAuthError(c, http.StatusServiceUnavailable, "gateway_not_initialized", "grpc gateway is not initialized")
		})
		r.NoMethod(func(c *gin.Context) {
			writeAdminAuthError(c, http.StatusServiceUnavailable, "gateway_not_initialized", "grpc gateway is not initialized")
		})
		return
	}

	wrapped := gin.WrapH(gateway)
	adminProtected := adminAuthMiddleware(tokens, wrapped)
	r.NoRoute(func(c *gin.Context) {
		if isAdminHTTPPath(c.Request.URL.Path) {
			forwardGateway(c, adminProtected)
			return
		}
		forwardGateway(c, wrapped)
	})
	r.NoMethod(func(c *gin.Context) {
		if isAdminHTTPPath(c.Request.URL.Path) {
			forwardGateway(c, adminProtected)
			return
		}
		forwardGateway(c, wrapped)
	})
}

func setupHTTPRouter(r *gin.Engine) {
	registerHTTPRoutes(r, gatewayHandler, adminTokenValidator)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("Access-Control-Allow-Origin", "*")
		headers.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		headers.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin, X-Requested-With")
		headers.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		headers.Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func forwardGateway(c *gin.Context, next gin.HandlerFunc) {
	// Gin enters custom NoRoute/NoMethod handlers with a preset 404 status.
	// Reset it so downstream gateway handlers can default to 200 on success.
	c.Status(http.StatusOK)
	next(c)
}

func adminAuthMiddleware(tokens service.AdminTokenStore, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAdminLoginPath(c.Request.URL.Path) {
			next(c)
			return
		}
		if tokens == nil {
			writeAdminAuthError(c, http.StatusServiceUnavailable, "admin_auth_store_unavailable", "admin auth token store is not initialized")
			return
		}

		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			writeAdminAuthError(c, http.StatusUnauthorized, "admin_auth_missing_token", "missing bearer token")
			return
		}
		if _, err := tokens.ValidateToken(c.Request.Context(), token); err != nil {
			if errors.Is(err, service.ErrAdminTokenNotFound) {
				writeAdminAuthError(c, http.StatusUnauthorized, "admin_auth_invalid_token", "invalid bearer token")
				return
			}
			writeAdminAuthError(c, http.StatusServiceUnavailable, "admin_auth_validation_failed", "admin auth validation failed")
			return
		}
		next(c)
	}
}

func isAdminHTTPPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/admin/")
}

func isAdminLoginPath(path string) bool {
	return path == "/api/v1/admin/auth:login"
}

func bearerToken(header string) string {
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

func gatewayHeaderMatcher(key string) (string, bool) {
	if strings.EqualFold(key, "Authorization") {
		return "authorization", true
	}
	return runtime.DefaultHeaderMatcher(key)
}

func writeAdminAuthError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, gin.H{
		"code":    code,
		"message": message,
	})
}
