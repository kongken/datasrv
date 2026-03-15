package internal

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	feedsv1 "github.com/kongken/datasrv/pkg/proto/feeds/v1"
	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const grpcGatewayEndpoint = "localhost:9090"

type gatewayRegistrar func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

var gatewayHandler http.Handler

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
	mux := runtime.NewServeMux()
	for _, register := range registrars {
		if err := register(ctx, mux, endpoint, opts); err != nil {
			return nil, err
		}
	}
	return mux, nil
}

func registerHTTPRoutes(r *gin.Engine, gateway http.Handler) {
	if gateway == nil {
		r.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "grpc gateway is not initialized"})
		})
		r.NoMethod(func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "grpc gateway is not initialized"})
		})
		return
	}

	wrapped := gin.WrapH(gateway)
	r.NoRoute(wrapped)
	r.NoMethod(wrapped)
}

func setupHTTPRouter(r *gin.Engine) {
	registerHTTPRoutes(r, gatewayHandler)
}
