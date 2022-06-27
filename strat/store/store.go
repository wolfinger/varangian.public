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

// Store interface used for implementing the Strategy store
type Store interface {
	GetStrat(ctx context.Context, id string) (*storage.Strat, error)
	ListStrats(ctx context.Context) ([]*storage.Strat, error)
	UpdateStrat(ctx context.Context, strat *storage.Strat, fieldMask []string) error
	CreateStrat(ctx context.Context, strat *storage.Strat) (*storage.Strat, error)
	DeleteStrat(ctx context.Context, id string) error
}

// NewStore encapsulates Strategy database operations
func NewStore(conn *pg.DB) Store {
	return &storeImpl{
		conn: conn,
	}
}

type storeImpl struct {
	conn *pg.DB
}

// GetStrat gets a strategy from the Strategy store
func (s *storeImpl) GetStrat(ctx context.Context, id string) (*storage.Strat, error) {
	// convert vxids to vids
	vid, err := vxid.Decode(id)
	if err != nil {
		return nil, err
	}

	var strat storage.Strat
	err = s.conn.ModelContext(ctx, &strat).Where("id = ?", vid).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "strat with id %s not found", id)
		}
		return nil, err
	}
	strat.Id = id

	// convert vids to vxids
	if strat.GetParentId() != "" {
		strat.ParentId, err = vxid.Encode(strat.GetParentId(), vxid.PfxMap.Organization)
		if err != nil {
			return nil, err
		}
	}

	return &strat, err
}

// ListStrats lists an array of strategies from the Strategy store
func (s *storeImpl) ListStrats(ctx context.Context) ([]*storage.Strat, error) {
	var strats []*storage.Strat
	err := s.conn.ModelContext(ctx, &strats).Select()
	if err != nil {
		return nil, fmt.Errorf("listing strats %w", err)
	}

	for _, strat := range strats {
		// convert vids to vxids
		strat.Id, err = vxid.Encode(strat.GetId(), vxid.PfxMap.Strategy)
		if err != nil {
			return nil, err
		}
		if strat.GetParentId() != "" {
			strat.ParentId, err = vxid.Encode(strat.GetParentId(), vxid.PfxMap.Strategy)
			if err != nil {
				return nil, err
			}
		}
	}

	return strats, nil
}

// UpdateStrat updates a strategy via the Strategy store
func (s *storeImpl) UpdateStrat(ctx context.Context, strat *storage.Strat, fieldMask []string) error {
	var err error
	tgtStrat := strat

	// copy over only the fields passed in from the field mask (if provided)
	if fieldMask != nil {
		// get original strat object to update
		tgtStrat, err = s.GetStrat(ctx, strat.GetId())
		if err != nil {
			return err
		}

		mask, err := fieldmask_utils.MaskFromPaths(fieldMask, casing.Camel)
		if err != nil {
			return err
		}
		fieldmask_utils.StructToStruct(mask, strat, tgtStrat)
	}

	// convert vxids to vids
	tgtStrat.Id, err = vxid.Decode(tgtStrat.GetId())
	if err != nil {
		return err
	}
	if tgtStrat.GetParentId() != "" {
		tgtStrat.ParentId, err = vxid.Decode(tgtStrat.GetParentId())
		if err != nil {
			return err
		}
	}

	// update strat in datastore
	_, err = s.conn.ModelContext(ctx, tgtStrat).WherePK().Update()
	if err != nil {
		return fmt.Errorf("update strat %s %w", strat.GetId(), err)
	}

	return nil
}

// CreateStrat creates a new Strategy via the Strategy store
func (s *storeImpl) CreateStrat(ctx context.Context, strat *storage.Strat) (*storage.Strat, error) {
	var err error

	// convert vxids to vids
	if strat.GetParentId() != "" {
		strat.ParentId, err = vxid.Decode(strat.GetParentId())
		if err != nil {
			return nil, err
		}
	}

	// create strat in datastore
	_, err = s.conn.ModelContext(ctx, strat).Insert()
	if err != nil {
		return nil, err
	}

	// convert vids to vxids
	strat.Id, err = vxid.Encode(strat.GetId(), vxid.PfxMap.Strategy)
	if err != nil {
		return nil, err
	}
	if strat.GetParentId() != "" {
		strat.ParentId, err = vxid.Encode(strat.GetParentId(), vxid.PfxMap.Strategy)
	}

	return strat, nil
}

// DeleteStrat removes a strategy from the Strategy store
func (s *storeImpl) DeleteStrat(ctx context.Context, id string) error {
	// convert vxid to vid
	vid, err := vxid.Decode(id)
	if err != nil {
		return err
	}

	if _, err = s.conn.ModelContext(ctx, (*storage.Strat)(nil)).Where("id = ?", vid).Delete(); err != nil {
		return fmt.Errorf("deleting strat %s %w", id, err)
	}

	return nil
}
