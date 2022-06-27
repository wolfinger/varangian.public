package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/go-pg/urlstruct"
	"github.com/wolfinger/varangian/generated/storage"
	"github.com/wolfinger/varangian/internal/casing"
	"github.com/wolfinger/varangian/internal/vxid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"
)

// Store interface used for implementing the Lot store
type Store interface {
	GetLot(ctx context.Context, id string, dt string) (*storage.Lot, error)
	ListLots(ctx context.Context, pageSize int32, pageToken string, filter string, orderBy string) ([]*storage.Lot, error)
	UpdateLot(ctx context.Context, lot *storage.Lot, fieldMask []string) error
	CreateLot(ctx context.Context, lot *storage.Lot) (*storage.Lot, error)
	DeleteLot(ctx context.Context, lot *storage.Lot) error

	GetLotBal(ctx context.Context, id string, dt string) (*storage.LotBal, error)
	ListLotBals(ctx context.Context, dt string, ids []string) ([]*storage.LotBal, error)
	UpdateLotBal(ctx context.Context, lotBal *storage.LotBal) error
	CreateLotBal(ctx context.Context, lotBal *storage.LotBal) error
	DeleteLotBal(ctx context.Context, dt string, ids []string) error
}

// NewStore encapsulates Lot database operations
func NewStore(conn *pg.DB) Store {
	return &storeImpl{
		conn: conn,
	}
}

type storeImpl struct {
	conn *pg.DB
}

// LotFilter provides custom filter for the Transaction store
type LotFilter struct {
	ID       []string
	SrcTxnID []string
	urlstruct.Pager
	/*
		InstID   string
		OrigDT   string
		OrigSize float64
		LeOrgID  string
		AcctID   string
	*/
}

func (f *LotFilter) query(q *orm.Query) (*orm.Query, error) {
	//q = q.Model((*storage.Lot)(nil))
	// q = q.Relation("Lot")

	// ID filters
	if f.ID != nil {
		vids, err := vxid.Decodes(f.ID)
		if err != nil {
			return nil, err
		}
		q.Where("id IN (?)", pg.In(vids))
	}

	// SrcTxnID filters
	if f.SrcTxnID != nil {
		vids, err := vxid.Decodes(f.SrcTxnID)
		if err != nil {
			return nil, err

		}
		q.Where("src_txn_id IN (?)", pg.In(vids))
	}

	return q, nil
}

// GetLot retrieves a lot from the Lot service
func (s *storeImpl) GetLot(ctx context.Context, id string, dt string) (*storage.Lot, error) {
	// convert vxid to vid
	vid, err := vxid.Decode(id)
	if err != nil {
		return nil, err
	}

	var lot storage.Lot
	err = s.conn.ModelContext(ctx, &lot).ColumnExpr("*, orig_dt::date").Where("id = ?", vid).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "lot with id %s not found", id)
		}
		return nil, err
	}
	lot.Id = id

	// convert vids to vxids
	if lot.GetInstId() != "" {
		lot.InstId, err = vxid.Encode(lot.GetInstId(), vxid.PfxMap.Instrument)
		if err != nil {
			return nil, err
		}
	}
	if lot.GetSrcTxnId() != "" {
		lot.SrcTxnId, err = vxid.Encode(lot.GetSrcTxnId(), vxid.PfxMap.Transaction)
		if err != nil {
			return nil, err
		}
	}
	if lot.GetLeOrgId() != "" {
		lot.LeOrgId, err = vxid.Encode(lot.GetLeOrgId(), vxid.PfxMap.Organization)
		if err != nil {
			return nil, err
		}
	}
	if lot.GetAcctId() != "" {
		lot.AcctId, err = vxid.Encode(lot.GetAcctId(), vxid.PfxMap.Account)
		if err != nil {
			return nil, err
		}
	}

	// get balance data associated with lot if date is passed in
	if dt != "" {
		// TODO: parse query string to be able to pull date ranges
		var lotBal storage.LotBal

		err = s.conn.ModelContext(ctx, &lotBal).ColumnExpr("lot_dt::date, lot_size, settled_size, unsettled_size").Where("lot_id = ?", vid).Where("lot_dt = ?", dt).Select()
		if err != nil {
			if err == pg.ErrNoRows {
				return nil, status.Errorf(codes.NotFound, "lot with id %s and date %s not found", id, dt)
			}
			return nil, err
		}
		lot.Bal = append(lot.Bal, &lotBal)
	}

	return &lot, err
}

// ListLots lists an array of lots from the Lot store
func (s *storeImpl) ListLots(ctx context.Context, pageSize int32, pageToken string, filter string, orderBy string) ([]*storage.Lot, error) {
	// TODO: revisit if sending a date in should pull the balances for that date too
	/*
		dtFlag := false
		if dt != "" {
			dtFlag = true
		}
	*/

	// decode json filter string into struct
	f := new(LotFilter)
	json.Unmarshal([]byte(filter), &f)

	var lots []*storage.Lot
	q := s.conn.ModelContext(ctx, &lots).Apply(f.query)
	err := q.Select()
	if err != nil {
		return nil, fmt.Errorf("listing lots: %w", err)
	}

	/*

		if dtFlag == false {
			err = s.conn.ModelContext(ctx, &lots).Select()
			if err != nil {
				return nil, fmt.Errorf("listing lots: %w", err)
			}
		} else {
			// err = s.conn.Model(&lots).Relation("Bal", func(q *orm.Query) (*orm.Query, error) { return q.Where("lot_dt = ?", dt), nil }).Select()
			err = s.conn.ModelContext(ctx, &lots).ColumnExpr("lot.*, lot.orig_dt::date").Join("INNER JOIN lot_bals AS bal ON bal.lot_id = lot.id").Where("bal.lot_dt = ?", dt).Select()
		}
		if err != nil {
			return nil, fmt.Errorf("listing lots: %w", err)
		}
	*/

	for _, lot := range lots {
		// convert vids to vxids
		lot.Id, err = vxid.Encode(lot.GetId(), vxid.PfxMap.Lot)
		if err != nil {
			return nil, err
		}
		if lot.GetInstId() != "" {
			lot.InstId, err = vxid.Encode(lot.GetInstId(), vxid.PfxMap.Instrument)
			if err != nil {
				return nil, err
			}
		}
		if lot.GetSrcTxnId() != "" {
			lot.SrcTxnId, err = vxid.Encode(lot.GetSrcTxnId(), vxid.PfxMap.Transaction)
			if err != nil {
				return nil, err
			}
		}
		if lot.GetLeOrgId() != "" {
			lot.LeOrgId, err = vxid.Encode(lot.GetLeOrgId(), vxid.PfxMap.Organization)
			if err != nil {
				return nil, err
			}
		}
		if lot.GetAcctId() != "" {
			lot.AcctId, err = vxid.Encode(lot.GetAcctId(), vxid.PfxMap.Account)
			if err != nil {
				return nil, err
			}
		}
	}

	return lots, nil
}

// UpdateLot updates a lot via the Lot store
func (s *storeImpl) UpdateLot(ctx context.Context, lot *storage.Lot, fieldMask []string) error {
	// TODO: rewrite to allow for update both at the same time
	var err error
	tgtLot := lot

	// determine if we're upating a lot's reference data or point-in-time (balance) data
	balFlag := false
	if len(lot.GetBal()) > 0 {
		balFlag = true
	}

	if balFlag == false {
		// copy over only the fields passed in from the field mask (if provided)
		if fieldMask != nil {
			// get original acct object to update
			tgtLot, err = s.GetLot(ctx, lot.GetId(), "")
			if err != nil {
				return err
			}

			mask, err := fieldmask_utils.MaskFromPaths(fieldMask, casing.Camel)
			if err != nil {
				return err
			}
			fieldmask_utils.StructToStruct(mask, lot, tgtLot)
		}

		// convert vxids to vids
		tgtLot.Id, err = vxid.Decode(tgtLot.GetId())
		if err != nil {
			return err
		}
		if tgtLot.GetInstId() != "" {
			tgtLot.InstId, err = vxid.Decode(tgtLot.GetInstId())
			if err != nil {
				return err
			}
		}
		if tgtLot.GetSrcTxnId() != "" {
			tgtLot.SrcTxnId, err = vxid.Decode(tgtLot.GetSrcTxnId())
			if err != nil {
				return err
			}
		}
		if tgtLot.GetLeOrgId() != "" {
			tgtLot.LeOrgId, err = vxid.Decode(tgtLot.GetLeOrgId())
			if err != nil {
				return err
			}
		}
		if tgtLot.GetAcctId() != "" {
			tgtLot.AcctId, err = vxid.Decode(tgtLot.GetAcctId())
			if err != nil {
				return err
			}
		}

		// update lot in datastore
		_, err = s.conn.ModelContext(ctx, tgtLot).WherePK().Update()
		if err != nil {
			return fmt.Errorf("update lot %s: %w", lot.Id, err)
		}
	} else {
		// update lot balance(s)
		lotBals := lot.GetBal()
		for _, lotBal := range lotBals {
			lotBal.LotId = lot.Id
			err := s.UpdateLotBal(ctx, lotBal)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// CreateLot creates a new lot via the Lot store
func (s *storeImpl) CreateLot(ctx context.Context, lot *storage.Lot) (*storage.Lot, error) {
	var err error

	// determine if we're inserting lot reference data or point-in-time data
	balFlag := false
	if len(lot.GetBal()) > 0 {
		balFlag = true
	}

	if balFlag == false {
		// save off vxids before converting them to vids to save some cycles
		var xLot storage.Lot
		xLot.InstId = lot.GetInstId()
		xLot.SrcTxnId = lot.GetSrcTxnId()
		xLot.LeOrgId = lot.GetLeOrgId()
		xLot.AcctId = lot.GetAcctId()

		// convert vxids to vids
		if lot.GetInstId() != "" {
			lot.InstId, err = vxid.Decode(lot.GetInstId())
			if err != nil {
				return nil, err
			}
		}
		if lot.GetSrcTxnId() != "" {
			lot.SrcTxnId, err = vxid.Decode(lot.GetSrcTxnId())
			if err != nil {
				return nil, err
			}
		}
		if lot.GetLeOrgId() != "" {
			lot.LeOrgId, err = vxid.Decode(lot.GetLeOrgId())
			if err != nil {
				return nil, err
			}
		}
		if lot.GetAcctId() != "" {
			lot.AcctId, err = vxid.Decode(lot.GetAcctId())
			if err != nil {
				return nil, err
			}
		}

		// add lot to datastore
		_, err = s.conn.ModelContext(ctx, lot).Insert()
		if err != nil {
			return nil, err
		}

		// generate initial balance for new lot
		// TODO: support settled vs. unsettled auto insert
		lotBal := &storage.LotBal{
			LotId:         lot.GetId(),
			LotDt:         lot.GetOrigDt(),
			LotSize:       lot.GetOrigSize(),
			SettledSize:   0,
			UnsettledSize: lot.GetOrigSize(),
		}
		_, err = s.conn.ModelContext(ctx, lotBal).Insert()
		if err != nil {
			return nil, err
		}

		// convert vids to vxids
		lot.Id, err = vxid.Encode(lot.Id, vxid.PfxMap.Lot)
		if err != nil {
			return nil, err
		}
		lot.InstId = xLot.GetInstId()
		lot.SrcTxnId = xLot.GetSrcTxnId()
		lot.LeOrgId = xLot.GetLeOrgId()
		lot.AcctId = xLot.GetAcctId()
	} else {
		// TODO: rewrite so new lots generate their initial lot bal here too
		// insert lot balances
		lot.Id, err = vxid.Decode(lot.GetId())
		if err != nil {
			return nil, err
		}

		lotBals := lot.GetBal()
		for _, lotBal := range lotBals {
			// TODO: ability to set initial lot balance to settled or unsettled
			lotBal.LotId = lot.GetId()
			lotBal.SettledSize = 0
			lotBal.UnsettledSize = lotBal.GetLotSize()
			// add lotbal to datastore
			// TODO: call createlotbal funciton instead of calling the insert?
			_, err = s.conn.ModelContext(ctx, lotBal).Insert()
			if err != nil {
				return nil, err
			}
		}

		// convert vids to vxids
		lot.Id, err = vxid.Encode(lot.Id, vxid.PfxMap.Lot)
		if err != nil {
			return nil, err
		}
	}

	return lot, nil
}

// DeleteLot removes a lot from the Lot store
func (s *storeImpl) DeleteLot(ctx context.Context, lot *storage.Lot) error {
	// convert vxid to vid
	vid, err := vxid.Decode(lot.GetId())
	if err != nil {
		return err
	}

	// determine if we're deleting lot refernece data or point-in-time data
	balFlag := false
	if len(lot.GetBal()) > 0 {
		balFlag = true
	}

	if balFlag == false {
		// delete any associated lot balances first
		if _, err = s.conn.ModelContext(ctx, (*storage.LotBal)(nil)).Where("lot_id = ?", vid).Delete(); err != nil {
			return fmt.Errorf("deleting %s from lot_bals: %w", lot.GetId(), err)
		}
		// delete the lot
		if _, err = s.conn.ModelContext(ctx, (*storage.Lot)(nil)).Where("id = ?", vid).Delete(); err != nil {
			return fmt.Errorf("deleting %s from lots: %w", lot.GetId(), err)
		}
	} else {
		lotBals := lot.GetBal()
		for _, lotBal := range lotBals {
			if _, err = s.conn.ModelContext(ctx, (*storage.LotBal)(nil)).Where("lot_id = ?", vid).Where("lot_dt = ?", lotBal.LotDt).Delete(); err != nil {
				return fmt.Errorf("deleting %s from lot_bals for dt %s: %w", lot.GetId(), lotBal.GetLotDt(), err)
			}
		}
	}

	return nil
}

// GetLotBal gets a lot balance for a given lot id and date
func (s *storeImpl) GetLotBal(ctx context.Context, id string, dt string) (*storage.LotBal, error) {
	// convert vxid to vid
	vid, err := vxid.Decode(id)
	if err != nil {
		return nil, err
	}

	var lotBal storage.LotBal
	err = s.conn.ModelContext(ctx, &lotBal).Where("lot_id = ?", vid).Where("lot_dt = ?", dt).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "lotBal with id %s not found", id)
		}
		return nil, err
	}
	lotBal.LotId = id
	lotBal.LotDt = dt

	return &lotBal, nil
}

// ListLotBals lists a set of lot balances for a given date and optional list of vxids
func (s *storeImpl) ListLotBals(ctx context.Context, dt string, ids []string) ([]*storage.LotBal, error) {
	var err error
	var vids []string
	var lotBals []*storage.LotBal

	idsFlag := false
	if ids != nil {
		idsFlag = true
		vids, err = vxid.Decodes(ids)
		if err != nil {
			return nil, err
		}
	}

	// execute select query based on if ids were passed in
	if idsFlag == false {
		err = s.conn.ModelContext(ctx, &lotBals).Where("lot_dt = ?", dt).Select()
	} else {
		err = s.conn.ModelContext(ctx, &lotBals).Where("lot_dt = ?", dt).Where("lot_id = ANY (?)", pg.In(vids)).Select()
	}
	if err != nil {
		return nil, fmt.Errorf("listing lot bals: %w", err)
	}

	for _, lotBal := range lotBals {
		// convert vid to vxid
		lotBal.LotId, err = vxid.Encode(lotBal.GetLotId(), vxid.PfxMap.Lot)
		if err != nil {
			return nil, err
		}
	}

	return lotBals, nil
}

// UpdateLotBal updates a lot balance for a given date
func (s *storeImpl) UpdateLotBal(ctx context.Context, lotBal *storage.LotBal) error {
	// update lot balance(s)
	var err error
	vxLotID := lotBal.GetLotId()
	lotBal.LotId, err = vxid.Decode(lotBal.GetLotId())
	if err != nil {
		return err
	}

	_, err = s.conn.ModelContext(ctx, lotBal).WherePK().Update()
	if err != nil {
		return fmt.Errorf("update lot %s on %s: %w", vxLotID, lotBal.LotDt, err)
	}

	return nil
}

// CreateLotBal creates a lot balance for a given date
func (s *storeImpl) CreateLotBal(ctx context.Context, lotBal *storage.LotBal) error {
	var err error
	// convert vxids to vids
	lotBal.LotId, err = vxid.Decode(lotBal.GetLotId())
	if err != nil {
		return err
	}

	// add lotbal to datastore
	_, err = s.conn.ModelContext(ctx, lotBal).Insert()
	if err != nil {
		return err
	}

	return nil
}

// DeleteLotBal deletes a lot balance
func (s *storeImpl) DeleteLotBal(ctx context.Context, dt string, ids []string) error {
	var err error
	var vids []string

	// convert vxids to vids
	if ids != nil {
		vids, err = vxid.Decodes(ids)
		if err != nil {
			return err
		}
	}

	if ids == nil {
		_, err = s.conn.ModelContext(ctx, (*storage.LotBal)(nil)).Where("lot_dt = ?", dt).Delete()
	} else {
		_, err = s.conn.ModelContext(ctx, (*storage.LotBal)(nil)).Where("lot_dt = ?", dt).Where("lot_id = ANY (?)", pg.In(vids)).Delete()
	}
	if err != nil {
		return fmt.Errorf("deleting from lot_bals where date is %s: %w", dt, err)
	}

	return nil
}
