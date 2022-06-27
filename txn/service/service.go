package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/wolfinger/varangian/generated/api/v1"
	"github.com/wolfinger/varangian/generated/storage"
	lotStore "github.com/wolfinger/varangian/lot/store"
	grpcPkg "github.com/wolfinger/varangian/pkg/grpc"
	txnStore "github.com/wolfinger/varangian/txn/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type txnType struct {
	Trade      string
	Settle     string
	Income     string
	Sweep      string
	Transfer   string
	Allocation string
}

type txnSubType struct {
	Trade struct {
		Buy      string
		Sell     string
		Reinvest string
	}
	Settle string
	Income struct {
		Dividend string
		Interest string
	}
	Sweep struct {
		In  string
		Out string
	}
	Transfer   string
	Allocation string
}

type txnState struct {
	Open      string
	Pending   string
	Processed string
}

var (
	// TxnType defines lists of transaction types supported
	TxnType = txnType{
		Trade:      "trade",
		Settle:     "settle",
		Income:     "income",
		Sweep:      "sweep",
		Transfer:   "xfer",
		Allocation: "allocation"}

	// TxnSubType defines lists of transaction subtypes supported
	TxnSubType = txnSubType{
		Trade: struct {
			Buy      string
			Sell     string
			Reinvest string
		}{
			Buy:      "buy",
			Sell:     "sell",
			Reinvest: "reinvest"},
		Settle: TxnType.Settle,
		Income: struct {
			Dividend string
			Interest string
		}{
			Dividend: "dividend",
			Interest: "interest"},
		Sweep: struct {
			In  string
			Out string
		}{
			In:  "in",
			Out: "out"},
		Transfer:   TxnType.Transfer,
		Allocation: TxnType.Allocation}

	// TxnState defines the list of transaction states supported
	TxnState = txnState{
		Open:      "open",
		Pending:   "pending",
		Processed: "processed"}
)

// Service interface used for implementing the Transaction service
type Service interface {
	v1.TxnServiceServer
	grpcPkg.Service
}

// NewService creates new Transaction service
func NewService(txnStore txnStore.Store, lotStore lotStore.Store) *TxnServiceImpl {
	return &TxnServiceImpl{
		txnStore: txnStore,
		lotStore: lotStore,
	}
}

// TxnServiceImpl data structure for implementing the Transaction service
type TxnServiceImpl struct {
	txnStore txnStore.Store
	lotStore lotStore.Store
}

// RegisterServer registers the Transaction service server
func (s *TxnServiceImpl) RegisterServer(server *grpc.Server) {
	v1.RegisterTxnServiceServer(server, s)
}

// RegisterHandler registers the Transaction handler service
func (s *TxnServiceImpl) RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterTxnServiceHandler(ctx, mux, conn)
}

// GetTxn gets a transaction from the Transaction service
func (s *TxnServiceImpl) GetTxn(ctx context.Context, request *v1.GetTxnRequest) (*v1.GetTxnResponse, error) {
	txn, err := s.txnStore.GetTxn(ctx, request.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.GetTxnResponse{
		Txn: txn,
	}, nil
}

// ListTxns lists an array of transacitons from the Transaction service
func (s *TxnServiceImpl) ListTxns(ctx context.Context, request *v1.ListTxnsRequest) (*v1.ListTxnsResponse, error) {
	// base64.RawURLEncoding.DecodeString(request.GetFilter()) -- implement for converting base64

	txns, err := s.txnStore.ListTxns(ctx, request.GetMaxPageSize(), request.GetPageToken(), request.GetFilter(), request.GetOrderBy())
	if err != nil {
		return nil, err
	}

	return &v1.ListTxnsResponse{
		Txns: txns,
	}, nil
}

// UpdateTxn updates a transaction via the Transaction service
func (s *TxnServiceImpl) UpdateTxn(ctx context.Context, request *v1.UpdateTxnRequest) (*v1.UpdateTxnResponse, error) {
	request.GetTxn().Id = request.GetId()

	if err := s.txnStore.UpdateTxn(ctx, request.GetTxn(), request.GetUpdateMask().GetPaths()); err != nil {
		return nil, err
	}

	return &v1.UpdateTxnResponse{}, nil
}

// CreateTxn creates a new transaction via the Transaction service
func (s *TxnServiceImpl) CreateTxn(ctx context.Context, request *v1.CreateTxnRequest) (*v1.CreateTxnResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "txn required in POST")
	}

	if request.GetTxn().GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "txn id is not expected in POST")
	}

	txn, err := s.txnStore.CreateTxn(ctx, request.GetTxn())
	if err != nil {
		return nil, err
	}

	return &v1.CreateTxnResponse{
		Txn: txn,
	}, nil
}

// DeleteTxn removes a transaction from the Transaction service
func (s *TxnServiceImpl) DeleteTxn(ctx context.Context, request *v1.DeleteTxnRequest) (*v1.DeleteTxnResponse, error) {
	if err := s.txnStore.DeleteTxn(ctx, request.GetId()); err != nil {
		return nil, err
	}

	return &v1.DeleteTxnResponse{}, nil
}

// ProcessTxn processes a transaction
func (s *TxnServiceImpl) ProcessTxn(ctx context.Context, request *v1.ProcessTxnRequest) (*v1.ProcessTxnResponse, error) {
	txn, err := s.txnStore.GetTxn(ctx, request.GetId())
	if err != nil {
		return nil, err
	}

	// only process open transactions
	if txn.State == TxnState.Open {
		switch txn.TxnType {
		// trade
		case TxnType.Trade:
			switch txn.TxnSubType {
			// buy
			case TxnSubType.Trade.Buy:
				var lot storage.Lot
				lot.InstId = txn.InstId
				lot.SrcTxnId = txn.Id
				lot.OrigDt = txn.TxnDt
				lot.OrigSize = txn.TxnSize

				// create new lot from buy transaction
				_, err = s.lotStore.CreateLot(ctx, &lot)
				if err != nil {
					return nil, fmt.Errorf("creating lot from processing txn %s: %w", request.GetId(), err)
				}
			// sell
			case TxnSubType.Trade.Sell:
				// determine lot list to sell against
				saleLotIDs := request.GetLotIds()
				if saleLotIDs == nil {
					// TODO: do stuff to find lots
				}
				// reduce the lot balances
				balRemaining := txn.GetTxnSize()
				for _, lotID := range saleLotIDs {
					lotBal, err := s.lotStore.GetLotBal(ctx, lotID, txn.TxnDt)
					if err != nil {
						return nil, err
					}
					var allocSize float64
					if balRemaining < lotBal.LotSize {
						allocSize = balRemaining
						lotBal.LotSize -= balRemaining
						lotBal.UnsettledSize -= balRemaining
						balRemaining = 0
					} else {
						allocSize = lotBal.LotSize
						balRemaining -= lotBal.LotSize
						lotBal.LotSize = 0
						lotBal.UnsettledSize = 0
					}
					err = s.lotStore.UpdateLotBal(ctx, lotBal)
					if err != nil {
						return nil, err
					}
					// generate allocating transaction
					var allocTxn storage.Txn
					allocTxn.TxnDt = txn.TxnDt
					allocTxn.SettleDt = txn.TxnDt
					allocTxn.TxnType = TxnType.Allocation
					allocTxn.TxnSize = allocSize
					// allocTxn.InstId = txn.InstId
					allocTxn.ParentId = txn.Id
					allocTxn.TgtLotId = lotID
					allocTxn.State = TxnState.Processed
					_, err = s.txnStore.CreateTxn(ctx, &allocTxn)
					if err != nil {
						return nil, err
					}

					// exit the loop once the size is fully allocated
					if balRemaining == 0 {
						break
					}
				}
			// reinvest
			case TxnSubType.Trade.Reinvest:
				// create new lot based on reinvestment
				var lot storage.Lot
				lot.InstId = txn.InstId
				lot.SrcTxnId = txn.Id
				lot.OrigDt = txn.TxnDt
				lot.OrigSize = txn.TxnSize

				// create lot
				reinvestLot, err := s.lotStore.CreateLot(ctx, &lot)
				if err != nil {
					return nil, fmt.Errorf("creating lot from processing txn %s: %w", request.GetId(), err)
				}

				// get lotBal and update to auto-settle
				reinvestLotBal, err := s.lotStore.GetLotBal(ctx, reinvestLot.GetId(), txn.GetSettleDt())
				if err != nil {
					return nil, err
				}

				reinvestLotBal.SettledSize = reinvestLotBal.LotSize
				reinvestLotBal.UnsettledSize = 0

				err = s.lotStore.UpdateLotBal(ctx, reinvestLotBal)
				if err != nil {
					return nil, err
				}

				// find funding lot (using src_lot_id)
				fundingLotBal, err := s.lotStore.GetLotBal(ctx, txn.GetSrcLotId(), txn.GetSettleDt())
				if err != nil {
					return nil, err
				}

				// error check if funding lot bal is what we expect
				if fundingLotBal.GetLotSize() != txn.GetSettleAmtNet() {
					return nil, fmt.Errorf("funding lot %s for reinvest txn %s not the same size", fundingLotBal.GetLotId(), txn.GetId())
				}

				// update funding lot to 0
				fundingLotBal.LotSize = 0
				fundingLotBal.SettledSize = 0
				fundingLotBal.UnsettledSize = 0
				err = s.lotStore.UpdateLotBal(ctx, fundingLotBal)
				if err != nil {
					return nil, err
				}
			}
			// generate payable/receivable for non-reinvestment trades
			if txn.TxnSubType != TxnSubType.Trade.Reinvest {
				var payRecLot storage.Lot
				payRecLot.InstId = txn.GetSettleAmtCcyId()
				payRecLot.SrcTxnId = txn.GetId()
				payRecLot.OrigDt = txn.GetTxnDt()
				payRecLot.OrigSize = txn.GetSettleAmtNet()
				_, err = s.lotStore.CreateLot(ctx, &payRecLot)
				if err != nil {
					return nil, fmt.Errorf("creating payable/receivable lot from processing txn %s: %w", request.GetId(), err)
				}
			}
		// settle
		case TxnType.Settle:
			// get the original txn for settlement
			origTxn, err := s.txnStore.GetTxn(ctx, txn.GetParentId())
			if err != nil {
				return nil, err
			}

			// get allocating txns for the settlement
			// TODO: lookup how to write the function to ignore pagination / sort fields
			filter := txnStore.TxnFilter{
				TxnType:  []string{TxnType.Allocation},
				ParentID: []string{origTxn.GetId()},
			}
			filterJSON, err := json.Marshal(filter)
			if err != nil {
				return nil, err
			}
			allocTxns, err := s.txnStore.ListTxns(ctx, 0, "", string(filterJSON), "")
			if err != nil {
				return nil, err
			}

			// calc total allocation size found
			allocTotTxnSize := 0.0
			for _, allocTxn := range allocTxns {
				allocTotTxnSize += allocTxn.TxnSize
			}

			// verify allocating txns total to expected settlement amount
			if origTxn.GetTxnSize() != allocTotTxnSize {
				return nil, fmt.Errorf("finding allocating txns; expecting %f, found %f", origTxn.GetTxnSize(), allocTotTxnSize)
			}

			// loop thru allocating txns to update lotbals
			for _, allocTxn := range allocTxns {
				lotBal, err := s.lotStore.GetLotBal(ctx, allocTxn.GetTgtLotId(), txn.GetSettleDt())
				if err != nil {
					return nil, err
				}

				// update the settled size (decrease for sells, increase for buys/reinvests)
				multiplier := 1.0
				if origTxn.TxnSubType == TxnSubType.Trade.Sell {
					multiplier = -1.0
				}
				settleSize := allocTxn.GetTxnSize() * multiplier
				lotBal.SettledSize += settleSize
				// update lot bal in the data store
				err = s.lotStore.UpdateLotBal(ctx, lotBal)
				if err != nil {
					return nil, err
				}
			}

			// find payable/receivable lot using src_txn_id in lot
			lotFilter := lotStore.LotFilter{
				SrcTxnID: []string{txn.GetParentId()},
			}
			lotFilterJSON, err := json.Marshal(lotFilter)
			if err != nil {
				return nil, err
			}
			payRecLots, err := s.lotStore.ListLots(ctx, 0, "", string(lotFilterJSON), "")
			if err != nil {
				return nil, err
			}
			if len(payRecLots) > 1 {
				return nil, fmt.Errorf("Found more than one payable/receivable processing txn: %s with parent id: %s", txn.GetId(), txn.GetParentId())
			}

			// update payable/receivable settle size which will implicitly turn it into a normal currency holding
			payRecLot := payRecLots[0]
			payRecLotBal, err := s.lotStore.GetLotBal(ctx, payRecLot.GetId(), txn.GetSettleDt())
			payRecLotBal.SettledSize = payRecLotBal.GetLotSize()
			payRecLotBal.UnsettledSize = 0
			err = s.lotStore.UpdateLotBal(ctx, payRecLotBal)
			if err != nil {
				return nil, err
			}
		// sweep
		case TxnType.Sweep:
			var sweepLotBalID string
			var cashLotBalID string

			// determine which txn lotbal id is the sweep and cash
			if txn.GetTxnSubType() == TxnSubType.Sweep.In {
				sweepLotBalID = txn.GetTgtLotId()
				cashLotBalID = txn.GetSrcLotId()
			} else {
				sweepLotBalID = txn.GetSrcLotId()
				cashLotBalID = txn.GetTgtLotId()
			}

			// get source lot id size, settled size, and unsettled size
			cashLotBal, err := s.lotStore.GetLotBal(ctx, cashLotBalID, txn.GetSettleDt())
			if err != nil {
				return nil, err
			}

			// if unsettled size is != zero, error out (can't sweep unsettled cash)
			if cashLotBal.GetUnsettledSize() != 0 {
				return nil, fmt.Errorf("source cash lot: %s has unsettled size while processing txn: %s", cashLotBalID, txn.GetId())
			}

			// get target lot id record
			sweepLotBal, err := s.lotStore.GetLotBal(ctx, sweepLotBalID, txn.GetSettleDt())
			if err != nil {
				return nil, err
			}

			// update target lot size and settled size based on source lot size
			sweepLotBal.LotSize += cashLotBal.LotSize
			sweepLotBal.SettledSize = sweepLotBal.LotSize
			s.lotStore.UpdateLotBal(ctx, sweepLotBal)

			// update source lot size and settled size to 0
			cashLotBal.LotSize = 0
			cashLotBal.SettledSize = 0
			s.lotStore.UpdateLotBal(ctx, cashLotBal)
		// income
		case TxnType.Income:
			switch txn.TxnSubType {
			// dividend
			case TxnSubType.Income.Dividend:
				var lot storage.Lot
				lot.InstId = txn.SettleAmtCcyId
				lot.SrcTxnId = txn.Id
				lot.OrigDt = txn.SettleDt
				lot.OrigSize = txn.TxnSize

				// create new lot from dividend transaction
				divLot, err := s.lotStore.CreateLot(ctx, &lot)
				if err != nil {
					return nil, fmt.Errorf("creating lot from processing txn %s: %w", request.GetId(), err)
				}

				// get the lotBal to update the settled/unsettled size
				divLotBal, err := s.lotStore.GetLotBal(ctx, divLot.GetId(), divLot.GetOrigDt())
				if err != nil {
					return nil, err
				}

				// update lotBal settled/unsettled amts (same day settle)
				divLotBal.SettledSize = divLotBal.GetLotSize()
				divLotBal.UnsettledSize = 0

				// update lotBal
				err = s.lotStore.UpdateLotBal(ctx, divLotBal)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// update transaction state to processed if all went well
	txn.State = TxnState.Processed
	err = s.txnStore.UpdateTxn(ctx, txn, nil)
	if err != nil {
		return nil, fmt.Errorf("updating transaction %s state to %s: %w", request.GetId(), TxnState.Processed, err)
	}

	return &v1.ProcessTxnResponse{
		Id:    request.GetId(),
		State: TxnState.Processed}, nil
}
