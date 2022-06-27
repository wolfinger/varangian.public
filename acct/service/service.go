package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	acctStore "github.com/wolfinger/varangian/acct/store"
	v1 "github.com/wolfinger/varangian/generated/api/v1"
	grpcPkg "github.com/wolfinger/varangian/pkg/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service interface used for implementing the Account service
type Service interface {
	v1.AcctServiceServer
	grpcPkg.Service
}

// NewService creates new Account service
func NewService(acctStore acctStore.Store) *AcctServiceImpl {
	return &AcctServiceImpl{
		acctStore: acctStore,
	}
}

// AcctServiceImpl data structure to implement the Account service
type AcctServiceImpl struct {
	acctStore acctStore.Store
}

// RegisterServer registers the Account service server
func (s *AcctServiceImpl) RegisterServer(server *grpc.Server) {
	v1.RegisterAcctServiceServer(server, s)
}

// RegisterHandler registers the Account service handler
func (s *AcctServiceImpl) RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterAcctServiceHandler(ctx, mux, conn)
}

// GetAcct gets an account from the Account service
func (s *AcctServiceImpl) GetAcct(ctx context.Context, request *v1.GetAcctRequest) (*v1.GetAcctResponse, error) {
	acct, err := s.acctStore.GetAcct(ctx, request.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.GetAcctResponse{
		Acct: acct,
	}, nil
}

// ListAccts lists an array of accounts from the Account service
func (s *AcctServiceImpl) ListAccts(ctx context.Context, request *v1.ListAcctsRequest) (*v1.ListAcctsResponse, error) {
	accts, err := s.acctStore.ListAccts(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.ListAcctsResponse{
		Accts: accts,
	}, nil
}

// UpdateAcct updates an account via the Account service
func (s *AcctServiceImpl) UpdateAcct(ctx context.Context, request *v1.UpdateAcctRequest) (*v1.UpdateAcctResponse, error) {
	request.GetAcct().Id = request.GetId()

	if err := s.acctStore.UpdateAcct(ctx, request.GetAcct(), request.GetUpdateMask().GetPaths()); err != nil {
		return nil, err
	}

	return &v1.UpdateAcctResponse{}, nil
}

// CreateAcct creates a new account via the Account service
func (s *AcctServiceImpl) CreateAcct(ctx context.Context, request *v1.CreateAcctRequest) (*v1.CreateAcctResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "acct required in POST")
	}
	if request.GetAcct().GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "acct id is not expected in POST")
	}

	acct, err := s.acctStore.CreateAcct(ctx, request.GetAcct())
	if err != nil {
		return nil, err
	}

	return &v1.CreateAcctResponse{
		Acct: acct,
	}, nil
}

// DeleteAcct removes an account from the Account service
func (s *AcctServiceImpl) DeleteAcct(ctx context.Context, request *v1.DeleteAcctRequest) (*v1.DeleteAcctResponse, error) {
	if err := s.acctStore.DeleteAcct(ctx, request.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteAcctResponse{}, nil
}
