package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/wolfinger/varangian/generated/api/v1"
	grpcPkg "github.com/wolfinger/varangian/pkg/grpc"
	portStore "github.com/wolfinger/varangian/port/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service interface used for implementing the Port service
type Service interface {
	v1.PortServiceServer
	grpcPkg.Service
}

// NewService creates new Port service
func NewService(portStore portStore.Store) *PortServiceImpl {
	return &PortServiceImpl{
		portStore: portStore,
	}
}

// PortServiceImpl data structure for implementing the Portfolio service
type PortServiceImpl struct {
	portStore portStore.Store
}

// RegisterServer registers the Portfolio service server
func (s *PortServiceImpl) RegisterServer(server *grpc.Server) {
	v1.RegisterPortServiceServer(server, s)
}

// RegisterHandler registers the Portfolio service handler
func (s *PortServiceImpl) RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterPortServiceHandler(ctx, mux, conn)
}

// GetPort gets a port from the Portfolio service
func (s *PortServiceImpl) GetPort(ctx context.Context, request *v1.GetPortRequest) (*v1.GetPortResponse, error) {
	port, err := s.portStore.GetPort(ctx, request.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.GetPortResponse{
		Port: port,
	}, nil
}

// ListPorts lists an array of ports from the Portfolio service
func (s *PortServiceImpl) ListPorts(ctx context.Context, empty *v1.ListPortsRequest) (*v1.ListPortsResponse, error) {
	ports, err := s.portStore.ListPorts(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.ListPortsResponse{
		Ports: ports,
	}, nil
}

// UpdatePort updates a portfolio via the Portfolio service
func (s *PortServiceImpl) UpdatePort(ctx context.Context, request *v1.UpdatePortRequest) (*v1.UpdatePortResponse, error) {
	if err := s.portStore.UpdatePort(ctx, request.GetPort(), request.GetUpdateMask().GetPaths()); err != nil {
		return nil, err
	}

	return &v1.UpdatePortResponse{}, nil
}

// CreatePort creates a new portfolio via the Portfolio service
func (s *PortServiceImpl) CreatePort(ctx context.Context, request *v1.CreatePortRequest) (*v1.CreatePortResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "port required in POST")
	}
	if request.GetPort().GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "port id is not expected in POST")
	}
	port, err := s.portStore.CreatePort(ctx, request.GetPort())
	if err != nil {
		return nil, err
	}

	return &v1.CreatePortResponse{
		Port: port,
	}, nil
}

// DeletePort removes a portfolio from the Portfolio service
func (s *PortServiceImpl) DeletePort(ctx context.Context, request *v1.DeletePortRequest) (*v1.DeletePortResponse, error) {
	if err := s.portStore.DeletePort(ctx, request.GetId()); err != nil {
		return nil, err
	}

	return &v1.DeletePortResponse{}, nil
}
