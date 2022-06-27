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

// Store interface used for implementing the Account store
type Store interface {
	GetAcct(ctx context.Context, id string) (*storage.Acct, error)
	ListAccts(ctx context.Context) ([]*storage.Acct, error)
	UpdateAcct(ctx context.Context, strat *storage.Acct, fieldMask []string) error
	CreateAcct(ctx context.Context, strat *storage.Acct) (*storage.Acct, error)
	DeleteAcct(ctx context.Context, id string) error
}

// NewStore encapsulates Account database operations
func NewStore(conn *pg.DB) Store {
	return &storeImpl{
		conn: conn,
	}
}

type storeImpl struct {
	conn *pg.DB
}

// GetAcct gets an account from the Account store
func (s *storeImpl) GetAcct(ctx context.Context, id string) (*storage.Acct, error) {
	vid, err := vxid.Decode(id)
	if err != nil {
		return nil, err
	}

	var acct storage.Acct
	err = s.conn.ModelContext(ctx, &acct).Where("id = ?", vid).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "account with id %s not found", id)
		}
		return nil, err
	}
	acct.Id = id

	// convert parent_id to vxid, if one exists
	if acct.GetParentId() != "" {
		acct.ParentId, err = vxid.Encode(acct.GetParentId(), vxid.PfxMap.Account)
		if err != nil {
			return nil, err
		}
	}

	return &acct, err
}

// ListAccts lists an array of accounts from the Account store
func (s *storeImpl) ListAccts(ctx context.Context) ([]*storage.Acct, error) {
	var accts []*storage.Acct
	err := s.conn.ModelContext(ctx, &accts).Select()
	if err != nil {
		return nil, fmt.Errorf("listing accounts %w", err)
	}

	for _, acct := range accts {
		acct.Id, err = vxid.Encode(acct.GetId(), vxid.PfxMap.Account)
		if err != nil {
			return nil, err
		}
		if acct.GetParentId() != "" {
			acct.ParentId, err = vxid.Encode(acct.GetParentId(), vxid.PfxMap.Account)
			if err != nil {
				return nil, err
			}
		}
	}

	return accts, nil
}

// UpdateAcct updates an account via the Account store
func (s *storeImpl) UpdateAcct(ctx context.Context, acct *storage.Acct, fieldMask []string) error {
	var err error
	tgtAcct := acct

	// copy over only the fields passed in from the field mask (if provided)
	if fieldMask != nil {
		// get original acct object to update
		tgtAcct, err = s.GetAcct(ctx, acct.GetId())
		if err != nil {
			return err
		}

		mask, err := fieldmask_utils.MaskFromPaths(fieldMask, casing.Camel)
		if err != nil {
			return err
		}
		fieldmask_utils.StructToStruct(mask, acct, tgtAcct)
	}

	// convert vxids to vids
	tgtAcct.Id, err = vxid.Decode(tgtAcct.GetId())
	if err != nil {
		return err
	}
	if tgtAcct.GetParentId() != "" {
		tgtAcct.ParentId, err = vxid.Decode(tgtAcct.GetParentId())
		if err != nil {
			return err
		}
	}

	// update account in datastore
	_, err = s.conn.ModelContext(ctx, tgtAcct).WherePK().Update()
	if err != nil {
		return fmt.Errorf("update account %s %w", acct.GetId(), err)
	}

	return nil
}

// CreateAcct creates a new account via the Account store
func (s *storeImpl) CreateAcct(ctx context.Context, acct *storage.Acct) (*storage.Acct, error) {
	var err error

	// decode vxids to vids
	if acct.GetParentId() != "" {
		acct.ParentId, err = vxid.Decode(acct.GetParentId())
		if err != nil {
			return nil, err
		}
	}

	// insert account into datastore
	_, err = s.conn.ModelContext(ctx, acct).Insert()
	if err != nil {
		return nil, err
	}

	// encode vids to vxids
	acct.Id, err = vxid.Encode(acct.GetId(), vxid.PfxMap.Account)
	if err != nil {
		return nil, err
	}
	if acct.GetParentId() != "" {
		acct.ParentId, err = vxid.Encode(acct.GetParentId(), vxid.PfxMap.Account)
		if err != nil {
			return nil, err
		}
	}

	return acct, nil
}

// DeleteAcct removes an account from the Account store
func (s *storeImpl) DeleteAcct(ctx context.Context, id string) error {
	vid, err := vxid.Decode(id)
	if err != nil {
		return err
	}

	// delete account from datastore
	if _, err = s.conn.ModelContext(ctx, (*storage.Acct)(nil)).Where("id = ?", vid).Delete(); err != nil {
		return fmt.Errorf("deleting acct %s %w", id, err)
	}

	return nil
}
