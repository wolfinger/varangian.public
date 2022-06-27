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

// Store interface used for implementing the Portfolio store
type Store interface {
	GetPort(ctx context.Context, id string) (*storage.Port, error)
	ListPorts(ctx context.Context) ([]*storage.Port, error)
	UpdatePort(ctx context.Context, strat *storage.Port, fieldMask []string) error
	CreatePort(ctx context.Context, strat *storage.Port) (*storage.Port, error)
	DeletePort(ctx context.Context, id string) error
}

// NewStore encapsulates Portfolio database operations
func NewStore(conn *pg.DB) Store {
	return &storeImpl{
		conn: conn,
	}
}

type storeImpl struct {
	conn *pg.DB
}

// GetPort gets a port from the Portfolio service
func (s *storeImpl) GetPort(ctx context.Context, id string) (*storage.Port, error) {
	// convert vxids to vids
	vid, err := vxid.Decode(id)
	if err != nil {
		return nil, err
	}

	var port storage.Port
	err = s.conn.ModelContext(ctx, &port).Where("id = ?", vid).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "port with id %s not found", id)
		}
		return nil, err
	}
	port.Id = id

	// convert vids to vxids
	if port.GetParentId() != "" {
		port.ParentId, err = vxid.Encode(port.GetParentId(), vxid.PfxMap.Portfolio)
		if err != nil {
			return nil, err
		}
	}

	return &port, nil
}

// ListPorts lists an array of ports from the Portfolio store
func (s *storeImpl) ListPorts(ctx context.Context) ([]*storage.Port, error) {
	var ports []*storage.Port
	err := s.conn.ModelContext(ctx, &ports).Select()
	if err != nil {
		return nil, fmt.Errorf("listing ports %w", err)
	}

	for _, port := range ports {
		// convert vids to vxids
		port.Id, err = vxid.Encode(port.GetId(), vxid.PfxMap.Portfolio)
		if err != nil {
			return nil, err
		}
		if port.GetParentId() != "" {
			port.ParentId, err = vxid.Encode(port.GetParentId(), vxid.PfxMap.Portfolio)
			if err != nil {
				return nil, err
			}
		}
	}

	return ports, nil
}

// UpdatePort updates a portfolio via the Portfolio store
func (s *storeImpl) UpdatePort(ctx context.Context, port *storage.Port, fieldMask []string) error {
	var err error
	tgtPort := port

	// copy over only the fields passed in from the field mask (if provided)
	if fieldMask != nil {

		// get original port object to update
		tgtPort, err = s.GetPort(ctx, port.GetId())
		if err != nil {
			return err
		}

		mask, err := fieldmask_utils.MaskFromPaths(fieldMask, casing.Camel)
		if err != nil {
			return err
		}
		fieldmask_utils.StructToStruct(mask, port, tgtPort)
	}

	// convert vxids to vids
	tgtPort.Id, err = vxid.Decode(tgtPort.GetId())
	if err != nil {
		return err
	}
	if port.GetParentId() != "" {
		tgtPort.ParentId, err = vxid.Decode(tgtPort.GetParentId())
		if err != nil {
			return err
		}
	}

	// update port in datastore
	_, err = s.conn.ModelContext(ctx, tgtPort).WherePK().Update()
	if err != nil {
		return fmt.Errorf("update port %s %w", port.GetId(), err)
	}

	return nil
}

// CreatePort creates a new portfolio via the Portfolio store
func (s *storeImpl) CreatePort(ctx context.Context, port *storage.Port) (*storage.Port, error) {
	var err error

	// convert vxids to vids
	if port.GetParentId() != "" {
		port.ParentId, err = vxid.Decode(port.GetParentId())
		if err != nil {
			return nil, err
		}
	}

	// insert portfolio into datastore
	_, err = s.conn.ModelContext(ctx, port).Insert()
	if err != nil {
		return nil, err
	}

	// convert vids to vxids
	port.Id, err = vxid.Encode(port.GetId(), vxid.PfxMap.Portfolio)
	if err != nil {
		return nil, err
	}
	if port.GetParentId() != "" {
		port.ParentId, err = vxid.Encode(port.GetParentId(), vxid.PfxMap.Portfolio)
	}

	return port, nil
}

// DeletePort removes a portfolio from the Portfolio store
func (s *storeImpl) DeletePort(ctx context.Context, id string) error {
	// convert vxid to vid
	vid, err := vxid.Decode(id)
	if err != nil {
		return err
	}

	if _, err = s.conn.ModelContext(ctx, (*storage.Port)(nil)).Where("id = ?", vid).Delete(); err != nil {
		return fmt.Errorf("deleting port %s %w", id, err)
	}

	return nil
}
