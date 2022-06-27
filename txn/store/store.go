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

// Store interface used for implementing the Transaction store
type Store interface {
	GetTxn(ctx context.Context, id string) (*storage.Txn, error)
	ListTxns(ctx context.Context, pageSize int32, pageToken string, filter string, orderBy string) ([]*storage.Txn, error)
	UpdateTxn(ctx context.Context, txn *storage.Txn, fieldMask []string) error
	CreateTxn(ctx context.Context, txn *storage.Txn) (*storage.Txn, error)
	DeleteTxn(ctx context.Context, id string) error
}

// NewStore encapsulates Transaction database operations
func NewStore(conn *pg.DB) Store {
	return &storeImpl{
		conn: conn,
	}
}

type storeImpl struct {
	conn *pg.DB
}

// TxnFilter provides custom filter for the Transaction store
type TxnFilter struct {
	ID         []string
	TxnType    []string
	TxnTypeNEQ []string
	ParentID   []string
	urlstruct.Pager
	/*
		TxnDt          string
		SettleDt       string
		TxnSubType     string
		TxnSize        float64
		InstID         string
		LotID          string
		State          string
		TradeAmtCcyID  string
		TradeAmtGross  float64
		TradeAmtNet    float64
		SettleAmtCcyID string
		SettleAmtGross float64
		SettleAmtNet   float64
	*/
}

func (f *TxnFilter) query(q *orm.Query) (*orm.Query, error) {
	//q = q.Model((*storage.Txn)(nil))
	// q = q.Relation("Txn")

	// ID filters
	if f.ID != nil {
		vids, err := vxid.Decodes(f.ID)
		if err != nil {
			return nil, err
		}
		q.Where("id IN (?)", pg.In(vids))
	}

	// TxnType filters
	if f.TxnType != nil {
		q.Where("txn_type IN (?)", pg.In(f.TxnType))
	}
	if f.TxnTypeNEQ != nil {
		q.Where("txn_type NOT IN (?)", pg.In(f.TxnTypeNEQ))
	}

	// ParentID filters
	if f.ParentID != nil {
		vids, err := vxid.Decodes(f.ParentID)
		if err != nil {
			return nil, err
		}
		q.Where("parent_id IN (?)", pg.In(vids))
	}

	return q, nil
}

// GetTxn gets a transaction from the Transaction store
func (s *storeImpl) GetTxn(ctx context.Context, id string) (*storage.Txn, error) {
	// convert vxid to vid
	vid, err := vxid.Decode(id)
	if err != nil {
		return nil, err
	}

	var txn storage.Txn
	err = s.conn.ModelContext(ctx, &txn).ColumnExpr("*, txn_dt::date, settle_dt::date").Where("id = ?", vid).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "transaction with id %s not found", id)
		}
		return nil, err
	}
	txn.Id = id

	// convert vids to vxids
	if txn.GetInstId() != "" {
		txn.InstId, err = vxid.Encode(txn.GetInstId(), vxid.PfxMap.Instrument)
		if err != nil {
			return nil, err
		}
	}
	if txn.GetParentId() != "" {
		txn.ParentId, err = vxid.Encode(txn.GetParentId(), vxid.PfxMap.Transaction)
		if err != nil {
			return nil, err
		}
	}
	if txn.GetSrcLotId() != "" {
		txn.SrcLotId, err = vxid.Encode(txn.GetSrcLotId(), vxid.PfxMap.Lot)
		if err != nil {
			return nil, err
		}
	}
	if txn.GetTgtLotId() != "" {
		txn.TgtLotId, err = vxid.Encode(txn.GetTgtLotId(), vxid.PfxMap.Lot)
		if err != nil {
			return nil, err
		}
	}
	if txn.GetTradeAmtCcyId() != "" {
		txn.TradeAmtCcyId, err = vxid.Encode(txn.GetTradeAmtCcyId(), vxid.PfxMap.Instrument)
		if err != nil {
			return nil, err
		}
	}
	if txn.GetSettleAmtCcyId() != "" {
		txn.SettleAmtCcyId, err = vxid.Encode(txn.GetSettleAmtCcyId(), vxid.PfxMap.Instrument)
		if err != nil {
			return nil, err
		}
	}

	return &txn, err
}

// ListTxns lists an array of transacitons from the Transaction store
func (s *storeImpl) ListTxns(ctx context.Context, pageSize int32, pageToken string, filter string, orderBy string) ([]*storage.Txn, error) {
	var txns []*storage.Txn

	// decode json filter string into struct
	f := new(TxnFilter)
	json.Unmarshal([]byte(filter), &f)

	q := s.conn.ModelContext(ctx, &txns).ColumnExpr("*, txn_dt::date, settle_dt::date").Apply(f.query)
	err := q.Select()
	if err != nil {
		return nil, fmt.Errorf("listing txns: %w", err)
	}

	for _, txn := range txns {
		// convert vids to vxids
		txn.Id, err = vxid.Encode(txn.GetId(), vxid.PfxMap.Transaction)
		if err != nil {
			return nil, err
		}
		if txn.GetInstId() != "" {
			txn.InstId, err = vxid.Encode(txn.GetInstId(), vxid.PfxMap.Instrument)
			if err != nil {
				return nil, err
			}
		}
		if txn.GetParentId() != "" {
			txn.ParentId, err = vxid.Encode(txn.GetParentId(), vxid.PfxMap.Transaction)
			if err != nil {
				return nil, err
			}
		}
		if txn.GetSrcLotId() != "" {
			txn.SrcLotId, err = vxid.Encode(txn.GetSrcLotId(), vxid.PfxMap.Lot)
			if err != nil {
				return nil, err
			}
		}
		if txn.GetTgtLotId() != "" {
			txn.TgtLotId, err = vxid.Encode(txn.GetTgtLotId(), vxid.PfxMap.Lot)
			if err != nil {
				return nil, err
			}
		}
		if txn.GetTradeAmtCcyId() != "" {
			txn.TradeAmtCcyId, err = vxid.Encode(txn.GetTradeAmtCcyId(), vxid.PfxMap.Instrument)
			if err != nil {
				return nil, err
			}
		}
		if txn.GetSettleAmtCcyId() != "" {
			txn.SettleAmtCcyId, err = vxid.Encode(txn.GetSettleAmtCcyId(), vxid.PfxMap.Instrument)
			if err != nil {
				return nil, err
			}
		}
	}

	return txns, nil
}

// UpdateTxn updates a transaction via the Transaction store
func (s *storeImpl) UpdateTxn(ctx context.Context, txn *storage.Txn, fieldMask []string) error {
	var err error
	tgtTxn := txn

	// if not performing a full replace, copy over only the fields passed in from the field mask
	if fieldMask != nil {
		// get original txn object to update
		tgtTxn, err = s.GetTxn(ctx, txn.GetId())
		if err != nil {
			return err
		}

		mask, err := fieldmask_utils.MaskFromPaths(fieldMask, casing.Camel)
		if err != nil {
			return err
		}
		fieldmask_utils.StructToStruct(mask, txn, tgtTxn)
	}

	// convert vxids to vids
	tgtTxn.Id, err = vxid.Decode(tgtTxn.GetId())
	if err != nil {
		return err
	}
	if tgtTxn.GetInstId() != "" {
		tgtTxn.InstId, err = vxid.Decode(tgtTxn.GetInstId())
		if err != nil {
			return err
		}
	}
	if tgtTxn.GetParentId() != "" {
		tgtTxn.ParentId, err = vxid.Decode(tgtTxn.GetParentId())
		if err != nil {
			return err
		}
	}
	if tgtTxn.GetSrcLotId() != "" {
		tgtTxn.SrcLotId, err = vxid.Decode(tgtTxn.GetSrcLotId())
		if err != nil {
			return err
		}
	}
	if tgtTxn.GetTgtLotId() != "" {
		tgtTxn.TgtLotId, err = vxid.Decode(tgtTxn.GetTgtLotId())
		if err != nil {
			return err
		}
	}
	if tgtTxn.GetTradeAmtCcyId() != "" {
		tgtTxn.TradeAmtCcyId, err = vxid.Decode(tgtTxn.GetTradeAmtCcyId())
		if err != nil {
			return err
		}
	}
	if tgtTxn.GetSettleAmtCcyId() != "" {
		tgtTxn.SettleAmtCcyId, err = vxid.Decode(tgtTxn.GetSettleAmtCcyId())
		if err != nil {
			return err
		}
	}

	// update txn in datastore
	_, err = s.conn.ModelContext(ctx, tgtTxn).WherePK().Update()
	if err != nil {
		return fmt.Errorf("update txn %s %w", txn.GetId(), err)
	}

	return nil
}

// CreateTxn creates a new transaction via the Transaction store
func (s *storeImpl) CreateTxn(ctx context.Context, txn *storage.Txn) (*storage.Txn, error) {
	var err error

	// save off vxids before converting them to vids to save some cycles
	var xTxn storage.Txn
	xTxn.InstId = txn.GetInstId()
	xTxn.ParentId = txn.GetParentId()
	xTxn.SrcLotId = txn.GetSrcLotId()
	xTxn.TgtLotId = txn.GetTgtLotId()
	xTxn.TradeAmtCcyId = txn.GetTradeAmtCcyId()
	xTxn.SettleAmtCcyId = txn.GetSettleAmtCcyId()

	// convert vxids to vids
	if txn.GetInstId() != "" {
		txn.InstId, err = vxid.Decode(txn.InstId)
		if err != nil {
			return nil, err
		}
	}
	if txn.GetParentId() != "" {
		txn.ParentId, err = vxid.Decode(txn.GetParentId())
		if err != nil {
			return nil, err
		}
	}
	if txn.GetSrcLotId() != "" {
		txn.SrcLotId, err = vxid.Decode(txn.GetSrcLotId())
		if err != nil {
			return nil, err
		}
	}
	if txn.GetTgtLotId() != "" {
		txn.TgtLotId, err = vxid.Decode(txn.GetTgtLotId())
		if err != nil {
			return nil, err
		}
	}
	if txn.GetTradeAmtCcyId() != "" {
		txn.TradeAmtCcyId, err = vxid.Decode(txn.GetTradeAmtCcyId())
		if err != nil {
			return nil, err
		}
	}
	if txn.GetSettleAmtCcyId() != "" {
		txn.SettleAmtCcyId, err = vxid.Decode(txn.GetSettleAmtCcyId())
		if err != nil {
			return nil, err
		}
	}

	// insert txn in datastore
	_, err = s.conn.ModelContext(ctx, txn).Insert()
	if err != nil {
		return nil, err
	}

	// convert vids to vxids
	txn.Id, err = vxid.Encode(txn.GetId(), vxid.PfxMap.Transaction)
	if err != nil {
		return nil, err
	}
	txn.InstId = xTxn.GetInstId()
	txn.ParentId = xTxn.GetParentId()
	txn.SrcLotId = xTxn.GetSrcLotId()
	txn.TgtLotId = xTxn.GetTgtLotId()
	txn.TradeAmtCcyId = xTxn.GetTradeAmtCcyId()
	txn.SettleAmtCcyId = xTxn.GetSettleAmtCcyId()

	return txn, nil
}

// DeleteTxn removes a transaction from the Transaction store
func (s *storeImpl) DeleteTxn(ctx context.Context, id string) error {
	// convert vxid to vid
	vid, err := vxid.Decode(id)
	if err != nil {
		return err
	}

	if _, err = s.conn.ModelContext(ctx, (*storage.Txn)(nil)).Where("id = ?", vid).Delete(); err != nil {
		return fmt.Errorf("deleting txn %s %w", id, err)
	}

	return nil
}
