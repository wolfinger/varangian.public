package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/wolfinger/varangian/generated/api/v1"
	grpcPkg "github.com/wolfinger/varangian/pkg/grpc"
	"github.com/wolfinger/varangian/pkg/version"
	"google.golang.org/grpc"
)

// Service implements the VersionService
type Service interface {
	v1.VersionServiceServer
	grpcPkg.Service
}

// NewService creates new version service
func NewService() Service {
	return &serviceImpl{}
}

type serviceImpl struct{}

func (s *serviceImpl) RegisterServer(server *grpc.Server) {
	v1.RegisterVersionServiceServer(server, s)
}

func (s *serviceImpl) RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterVersionServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) GetVersion(_ context.Context, _ *v1.VersionRequest) (*v1.VersionResponse, error) {
	return &v1.VersionResponse{
		Version: version.Version,
	}, nil
}
