package store

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/wolfinger/varangian/generated/storage"
	"github.com/wolfinger/varangian/internal/casing"
	"github.com/wolfinger/varangian/internal/vxid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"
)

// Store interface used for implementing the Organization store
type Store interface {
	GetOrg(ctx context.Context, id string) (*storage.Org, error)
	ListOrgs(ctx context.Context) ([]*storage.Org, error)
	UpdateOrg(ctx context.Context, strat *storage.Org, fieldMask []string) error
	CreateOrg(ctx context.Context, strat *storage.Org) (*storage.Org, error)
	DeleteOrg(ctx context.Context, id string) error
}

// NewStore encapsulates Organization database operations
func NewStore(conn *pg.DB) Store {
	return &storeImpl{
		conn: conn,
	}
}

type storeImpl struct {
	conn *pg.DB
}

// GetOrg gets an organization from the Organization store
func (s *storeImpl) GetOrg(ctx context.Context, id string) (*storage.Org, error) {
	// convert vxids to vids
	vid, err := vxid.Decode(id)
	if err != nil {
		return nil, err
	}

	var org storage.Org
	err = s.conn.ModelContext(ctx, &org).Where("id = ?", vid).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "org with id %s not found", id)
		}
		return nil, err
	}
	org.Id = id

	// convert vids to vxids
	if org.ParentId != "" {
		org.ParentId, err = vxid.Encode(org.ParentId, vxid.PfxMap.Organization)
		if err != nil {
			return nil, err
		}
	}

	return &org, err
}

// ListOrgs lists an array of organizations from the Organization store
func (s *storeImpl) ListOrgs(ctx context.Context) ([]*storage.Org, error) {
	var orgs []*storage.Org
	err := s.conn.ModelContext(ctx, &orgs).Select()

	for _, org := range orgs {
		org.Id, err = vxid.Encode(org.Id, vxid.PfxMap.Organization)
		if err != nil {
			return nil, err
		}
		// convert vids to vxids
		if org.ParentId != "" {
			org.ParentId, err = vxid.Encode(org.ParentId, vxid.PfxMap.Organization)
			if err != nil {
				return nil, err
			}
		}
	}

	return orgs, nil
}

// UpdateOrg updates an organization via the Organization store
func (s *storeImpl) UpdateOrg(ctx context.Context, org *storage.Org, fieldMask []string) error {
	var err error
	tgtOrg := org

	// copy over only the fields passed in from the field mask (if provided)
	if fieldMask != nil {
		// get original org object to update
		tgtOrg, err = s.GetOrg(ctx, org.GetId())
		if err != nil {
			return err
		}

		mask, err := fieldmask_utils.MaskFromPaths(fieldMask, casing.Camel)
		if err != nil {
			return err
		}
		fieldmask_utils.StructToStruct(mask, org, tgtOrg)
	}

	// convert vxids to vids
	tgtOrg.Id, err = vxid.Decode(tgtOrg.GetId())
	if err != nil {
		return err
	}
	if tgtOrg.GetParentId() != "" {
		tgtOrg.ParentId, err = vxid.Decode(tgtOrg.GetParentId())
		if err != nil {
			return err
		}
	}

	// update org in the datastore
	_, err = s.conn.ModelContext(ctx, tgtOrg).WherePK().Update()
	if err != nil {
		return fmt.Errorf("update org %s %w", org.GetId(), err)
	}

	return nil
}

// CreateOrg creats a new organization via the Organization store
func (s *storeImpl) CreateOrg(ctx context.Context, org *storage.Org) (*storage.Org, error) {
	var err error

	// convert vxids to vids
	if org.GetParentId() != "" {
		org.ParentId, err = vxid.Decode(org.GetParentId())
		if err != nil {
			return nil, err
		}
	}

	// insert org into datastore
	_, err = s.conn.ModelContext(ctx, org).Insert()
	if err != nil {
		return nil, err
	}

	// convert vids to vxids
	org.Id, err = vxid.Encode(org.GetId(), vxid.PfxMap.Organization)
	if err != nil {
		return nil, err
	}
	if org.GetParentId() != "" {
		org.ParentId, err = vxid.Encode(org.GetParentId(), vxid.PfxMap.Organization)
	}

	return org, nil
}

// DeleteOrg removes an organization from the Organization store
func (s *storeImpl) DeleteOrg(ctx context.Context, id string) error {
	vid, err := vxid.Decode(id)
	if err != nil {
		return err
	}

	if _, err = s.conn.ModelContext(ctx, (*storage.Org)(nil)).Where("id = ?", vid).Delete(); err != nil {
		return fmt.Errorf("deleting org %s %w", id, err)
	}

	return nil
}
