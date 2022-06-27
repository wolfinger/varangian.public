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

// Store interface used for implementing the Instrument store
type Store interface {
	GetInst(ctx context.Context, id string) (*storage.Inst, error)
	ListInsts(ctx context.Context) ([]*storage.Inst, error)
	UpdateInst(ctx context.Context, inst *storage.Inst, fieldMask []string) error
	CreateInst(ctx context.Context, inst *storage.Inst) (*storage.Inst, error)
	DeleteInst(ctx context.Context, id string) error
}

// NewStore encapsulates Instrument database operations
func NewStore(conn *pg.DB) Store {
	return &storeImpl{
		conn: conn,
	}
}

type storeImpl struct {
	conn *pg.DB
}

// GetInst gets an instrument from the Instrument store
func (s *storeImpl) GetInst(ctx context.Context, id string) (*storage.Inst, error) {
	// convert vxids to vids
	vid, err := vxid.Decode(id)
	if err != nil {
		return nil, err
	}

	var inst storage.Inst
	err = s.conn.ModelContext(ctx, &inst).Where("id = ?", vid).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "instrument with id %s not found", id)
		}
		return nil, err
	}
	inst.Id = id

	// convert vids to vxids
	if inst.GetProxyInst() != "" {
		inst.ProxyInst, err = vxid.Encode(inst.GetProxyInst(), vxid.PfxMap.Instrument)
		if err != nil {
			return nil, err
		}
	}

	return &inst, err
}

// ListInsts lists an array of instruments from the Instrument store
func (s *storeImpl) ListInsts(ctx context.Context) ([]*storage.Inst, error) {
	var insts []*storage.Inst
	err := s.conn.ModelContext(ctx, &insts).Select()
	if err != nil {
		return nil, fmt.Errorf("listing instruments %w", err)
	}

	for _, inst := range insts {
		// convert vids to vxids
		inst.Id, err = vxid.Encode(inst.GetId(), vxid.PfxMap.Instrument)
		if err != nil {
			return nil, err
		}
		if inst.GetProxyInst() != "" {
			inst.ProxyInst, err = vxid.Encode(inst.GetProxyInst(), vxid.PfxMap.Instrument)
			if err != nil {
				return nil, err
			}
		}
	}

	return insts, nil
}

// UpdateInst updates an instrument via the Instrument store
func (s *storeImpl) UpdateInst(ctx context.Context, inst *storage.Inst, fieldMask []string) error {
	var err error
	tgtInst := inst

	// copy over only the fields passed in from the field mask (if provided)
	if fieldMask != nil {
		// get original inst object to update
		tgtInst, err = s.GetInst(ctx, inst.GetId())
		if err != nil {
			return err
		}

		// copy over only the fields passed in from the field mask
		mask, err := fieldmask_utils.MaskFromPaths(fieldMask, casing.Camel)
		if err != nil {
			return err
		}
		fieldmask_utils.StructToStruct(mask, inst, tgtInst)
	}

	// convert vxids to vids
	tgtInst.Id, err = vxid.Decode(tgtInst.GetId())
	if err != nil {
		return err
	}
	if tgtInst.GetProxyInst() != "" {
		tgtInst.ProxyInst, err = vxid.Decode(tgtInst.GetProxyInst())
		if err != nil {
			return err
		}
	}

	// update instrument in datastore
	_, err = s.conn.ModelContext(ctx, tgtInst).WherePK().Update()
	if err != nil {
		return fmt.Errorf("update inst %s %w", tgtInst.GetId(), err)
	}

	return nil
}

// CreateInst creates a new instrument via the Instrument store
func (s *storeImpl) CreateInst(ctx context.Context, inst *storage.Inst) (*storage.Inst, error) {
	var err error

	// convert vxids to vids
	if inst.GetProxyInst() != "" {
		inst.ProxyInst, err = vxid.Decode(inst.GetProxyInst())
		if err != nil {
			return nil, err
		}
	}

	// create inst in datastore
	_, err = s.conn.ModelContext(ctx, inst).Insert()
	if err != nil {
		return nil, err
	}

	// convert vids to vxids
	inst.Id, err = vxid.Encode(inst.GetId(), vxid.PfxMap.Instrument)
	if err != nil {
		return nil, err
	}
	if inst.GetProxyInst() != "" {
		inst.ProxyInst, err = vxid.Encode(inst.GetProxyInst(), vxid.PfxMap.Instrument)
	}

	return inst, nil
}

// DeleteInst removes an instrument from the Instrument store
func (s *storeImpl) DeleteInst(ctx context.Context, id string) error {
	// convert vxid to vid
	vid, err := vxid.Decode(id)
	if err != nil {
		return err
	}

	if _, err = s.conn.ModelContext(ctx, (*storage.Inst)(nil)).Where("id = ?", vid).Delete(); err != nil {
		return fmt.Errorf("delete inst %s %w", id, err)
	}

	return nil
}
