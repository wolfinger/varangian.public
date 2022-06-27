package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/wolfinger/varangian/generated/api/v1"
	orgStore "github.com/wolfinger/varangian/org/store"
	grpcPkg "github.com/wolfinger/varangian/pkg/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service interface used for implementing the Organization service
type Service interface {
	v1.OrgServiceServer
	grpcPkg.Service
}

// NewService creates new Organization service
func NewService(orgStore orgStore.Store) *OrgServiceImpl {
	return &OrgServiceImpl{
		orgStore: orgStore,
	}
}

// OrgServiceImpl data structure for implementing the Organization service
type OrgServiceImpl struct {
	orgStore orgStore.Store
}

// RegisterServer registers the Organization service server
func (s *OrgServiceImpl) RegisterServer(server *grpc.Server) {
	v1.RegisterOrgServiceServer(server, s)
}

// RegisterHandler registers the Organization service handler
func (s *OrgServiceImpl) RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterOrgServiceHandler(ctx, mux, conn)
}

// GetOrg gets an organization from the Organization service
func (s *OrgServiceImpl) GetOrg(ctx context.Context, request *v1.GetOrgRequest) (*v1.GetOrgResponse, error) {
	org, err := s.orgStore.GetOrg(ctx, request.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.GetOrgResponse{
		Org: org,
	}, nil
}

// ListOrgs lists an array of organizations from the Organization service
func (s *OrgServiceImpl) ListOrgs(ctx context.Context, request *v1.ListOrgsRequest) (*v1.ListOrgsResponse, error) {
	orgs, err := s.orgStore.ListOrgs(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ListOrgsResponse{
		Orgs: orgs,
	}, nil
}

// UpdateOrg updates an organization via the Organization service
func (s *OrgServiceImpl) UpdateOrg(ctx context.Context, request *v1.UpdateOrgRequest) (*v1.UpdateOrgResponse, error) {
	request.GetOrg().Id = request.GetId()

	if err := s.orgStore.UpdateOrg(ctx, request.GetOrg(), request.GetUpdateMask().GetPaths()); err != nil {
		return nil, err
	}

	return &v1.UpdateOrgResponse{}, nil
}

// CreateOrg creats a new organization via the Organization service
func (s *OrgServiceImpl) CreateOrg(ctx context.Context, request *v1.CreateOrgRequest) (*v1.CreateOrgResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "org required in POST")
	}

	if request.GetOrg().GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "org id is not expected in POST")
	}

	org, err := s.orgStore.CreateOrg(ctx, request.GetOrg())
	if err != nil {
		return nil, err
	}

	return &v1.CreateOrgResponse{
		Org: org,
	}, nil
}

// DeleteOrg removes an organization from the Organization service
func (s *OrgServiceImpl) DeleteOrg(ctx context.Context, request *v1.DeleteOrgRequest) (*v1.DeleteOrgResponse, error) {
	if err := s.orgStore.DeleteOrg(ctx, request.GetId()); err != nil {
		return nil, err
	}

	return &v1.DeleteOrgResponse{}, nil
}
