package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/wolfinger/varangian/generated/api/v1"
	grpcPkg "github.com/wolfinger/varangian/pkg/grpc"
	stratStore "github.com/wolfinger/varangian/strat/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service interface used for implementing the Strategy service
type Service interface {
	v1.StratServiceServer
	grpcPkg.Service
}

// NewService creates new Strategy service
func NewService(stratStore stratStore.Store) *StratServiceImpl {
	return &StratServiceImpl{
		stratStore: stratStore,
	}
}

// StratServiceImpl data structure used for implementing the Strategy service
type StratServiceImpl struct {
	stratStore stratStore.Store
}

// RegisterServer registers the Strategy service server
func (s *StratServiceImpl) RegisterServer(server *grpc.Server) {
	v1.RegisterStratServiceServer(server, s)
}

// RegisterHandler registers the Strategy service handler
func (s *StratServiceImpl) RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterStratServiceHandler(ctx, mux, conn)
}

// GetStrat gets a strategy from the Strategy service
func (s *StratServiceImpl) GetStrat(ctx context.Context, request *v1.GetStratRequest) (*v1.GetStratResponse, error) {
	strat, err := s.stratStore.GetStrat(ctx, request.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.GetStratResponse{
		Strat: strat,
	}, nil
}

// ListStrats lists an array of strategies from the Strategy service
func (s *StratServiceImpl) ListStrats(ctx context.Context, request *v1.ListStratsRequest) (*v1.ListStratsResponse, error) {
	strats, err := s.stratStore.ListStrats(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.ListStratsResponse{
		Strats: strats,
	}, nil
}

// UpdateStrat updates a strategy via the Strategy service
func (s *StratServiceImpl) UpdateStrat(ctx context.Context, request *v1.UpdateStratRequest) (*v1.UpdateStratResponse, error) {
	if err := s.stratStore.UpdateStrat(ctx, request.GetStrat(), request.GetUpdateMask().GetPaths()); err != nil {
		return nil, err
	}

	return &v1.UpdateStratResponse{}, nil
}

// CreateStrat creates a new Strategy via the Strategy service
func (s *StratServiceImpl) CreateStrat(ctx context.Context, request *v1.CreateStratRequest) (*v1.CreateStratResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "strat required in POST")
	}

	if request.GetStrat().GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "strat id is not expected in POST")
	}

	strat, err := s.stratStore.CreateStrat(ctx, request.GetStrat())
	if err != nil {
		return nil, err
	}

	return &v1.CreateStratResponse{
		Strat: strat,
	}, nil
}

// DeleteStrat removes a strategy from the Strategy service
func (s *StratServiceImpl) DeleteStrat(ctx context.Context, request *v1.DeleteStratRequest) (*v1.DeleteStratResponse, error) {
	if err := s.stratStore.DeleteStrat(ctx, request.GetId()); err != nil {
		return nil, err
	}

	return &v1.DeleteStratResponse{}, nil
}
