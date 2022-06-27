package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/wolfinger/varangian/generated/api/v1"
	instStore "github.com/wolfinger/varangian/inst/store"
	grpcPkg "github.com/wolfinger/varangian/pkg/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service interface used for implementing the Instrument service
type Service interface {
	v1.InstServiceServer
	grpcPkg.Service
}

// NewService creates new Instrument service
func NewService(instStore instStore.Store) *InstServiceImpl {
	return &InstServiceImpl{
		instStore: instStore,
	}
}

// InstServiceImpl data structure for implementing the Instrument service
type InstServiceImpl struct {
	instStore instStore.Store
}

// RegisterServer registers the Instrument Service server
func (s *InstServiceImpl) RegisterServer(server *grpc.Server) {
	v1.RegisterInstServiceServer(server, s)
}

// RegisterHandler registers the Instrument service handler
func (s *InstServiceImpl) RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterInstServiceHandler(ctx, mux, conn)
}

// GetInst gets an instrument from the Instrument service
func (s *InstServiceImpl) GetInst(ctx context.Context, request *v1.GetInstRequest) (*v1.GetInstResponse, error) {
	inst, err := s.instStore.GetInst(ctx, request.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.GetInstResponse{
		Inst: inst,
	}, nil
}

// ListInsts lists an array of instruments from the Instrument service
func (s *InstServiceImpl) ListInsts(ctx context.Context, request *v1.ListInstsRequest) (*v1.ListInstsResponse, error) {
	insts, err := s.instStore.ListInsts(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.ListInstsResponse{
		Insts: insts,
	}, nil
}

// UpdateInst updates an instrument via the Instrument service
func (s *InstServiceImpl) UpdateInst(ctx context.Context, request *v1.UpdateInstRequest) (*v1.UpdateInstResponse, error) {
	request.GetInst().Id = request.GetId()

	if err := s.instStore.UpdateInst(ctx, request.GetInst(), request.GetUpdateMask().GetPaths()); err != nil {
		return nil, err
	}

	return &v1.UpdateInstResponse{}, nil
}

// CreateInst creates a new instrument via the Instrument service
func (s *InstServiceImpl) CreateInst(ctx context.Context, request *v1.CreateInstRequest) (*v1.CreateInstResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "instrument required in POST")
	}
	if request.GetInst().GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "instrument id is not expected in POST")
	}

	inst, err := s.instStore.CreateInst(ctx, request.GetInst())
	if err != nil {
		return nil, err
	}

	return &v1.CreateInstResponse{
		Inst: inst,
	}, nil
}

// DeleteInst removes an instrument from the Instrument service
func (s *InstServiceImpl) DeleteInst(ctx context.Context, request *v1.DeleteInstRequest) (*v1.DeleteInstResponse, error) {
	if err := s.instStore.DeleteInst(ctx, request.GetId()); err != nil {
		return nil, err
	}

	return &v1.DeleteInstResponse{}, nil
}
