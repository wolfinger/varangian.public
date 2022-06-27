package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/wolfinger/varangian/generated/api/v1"
	"github.com/wolfinger/varangian/generated/storage"
	lotStore "github.com/wolfinger/varangian/lot/store"
	grpcPkg "github.com/wolfinger/varangian/pkg/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service interface used for implementing the Lot service
type Service interface {
	v1.LotServiceServer
	grpcPkg.Service
}

// NewService creates new Lot service
func NewService(lotStore lotStore.Store) *LotServiceImpl {
	return &LotServiceImpl{
		lotStore: lotStore,
	}
}

// LotServiceImpl data structure for the implementing the Lot service
type LotServiceImpl struct {
	lotStore lotStore.Store
}

// RegisterServer registers the Lot service server
func (s *LotServiceImpl) RegisterServer(server *grpc.Server) {
	v1.RegisterLotServiceServer(server, s)
}

// RegisterHandler registers the Lot service Handler
func (s *LotServiceImpl) RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterLotServiceHandler(ctx, mux, conn)
}

// GetLot retrieves a lot from the Lot service
func (s *LotServiceImpl) GetLot(ctx context.Context, request *v1.GetLotRequest) (*v1.GetLotResponse, error) {
	lot, err := s.lotStore.GetLot(ctx, request.GetId(), request.GetDt())
	if err != nil {
		return nil, err
	}

	return &v1.GetLotResponse{
		Lot: lot,
	}, nil
}

// ListLots lists an array of lots from the Lot service
func (s *LotServiceImpl) ListLots(ctx context.Context, request *v1.ListLotsRequest) (*v1.ListLotsResponse, error) {
	lots, err := s.lotStore.ListLots(ctx, request.GetMaxPageSize(), request.GetPageToken(), request.GetFilter(), request.GetOrderBy())
	if err != nil {
		return nil, err
	}

	return &v1.ListLotsResponse{
		Lots: lots,
	}, nil
}

// UpdateLot updates a lot via the Lot service
func (s *LotServiceImpl) UpdateLot(ctx context.Context, request *v1.UpdateLotRequest) (*v1.UpdateLotResponse, error) {
	// TODO: rewrite to allow for update both at the same time
	request.GetLot().Id = request.GetId()

	if err := s.lotStore.UpdateLot(ctx, request.GetLot(), request.GetUpdateMask().GetPaths()); err != nil {
		return nil, err
	}

	return &v1.UpdateLotResponse{}, nil
}

// CreateLot creates a new lot via the Lot service
func (s *LotServiceImpl) CreateLot(ctx context.Context, request *v1.CreateLotRequest) (*v1.CreateLotResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "lot required in POST")
	}

	if (request.GetLot().GetId() != "") && (len(request.GetLot().GetBal()) == 0) {
		return nil, status.Error(codes.InvalidArgument, "lot id is not expected in POST")
	}

	lot, err := s.lotStore.CreateLot(ctx, request.GetLot())
	if err != nil {
		return nil, err
	}

	return &v1.CreateLotResponse{
		Lot: lot,
	}, nil
}

// DeleteLot removes a lot from the Lot service
func (s *LotServiceImpl) DeleteLot(ctx context.Context, request *v1.DeleteLotRequest) (*v1.DeleteLotResponse, error) {
	// delete lot from store
	var delLot storage.Lot
	delLot.Id = request.GetId()
	if err := s.lotStore.DeleteLot(ctx, &delLot); err != nil {
		return nil, err
	}

	return &v1.DeleteLotResponse{}, nil
}

// RollLots rolls forward or back one or more lots from one day to the next
func (s *LotServiceImpl) RollLots(ctx context.Context, request *v1.RollLotsRequest) (*v1.RollLotsResponse, error) {
	var err error

	if request.GetDt() == "" {
		return nil, status.Error(codes.InvalidArgument, "roll date expected in POST")
	}

	direction := "forward"
	if request.GetDirection() != "" {
		direction = request.GetDirection()
	}

	if direction == "back" {
		err = s.lotStore.DeleteLotBal(ctx, request.GetDt(), request.GetLots())
		if err != nil {
			return nil, err
		}
	} else {
		lotBals, err := s.lotStore.ListLotBals(ctx, request.GetDt(), request.GetLots())
		if err != nil {
			return nil, err
		}

		// increment date by one day
		nextDT, err := time.Parse("2006-01-02", request.GetDt())
		if err != nil {
			return nil, err
		}
		nextDT = nextDT.AddDate(0, 0, 1)
		nextDtStr := nextDT.Format(time.RFC3339)

		// update lot balances with new date and insert into store
		// TODO: a sql select into would be more efficient, but think this is needed
		//       to abstract away the data back end. figure out if there is a better
		//       way to handle at the service level while still abstracting backend
		for _, lotBal := range lotBals {
			// only roll lot if all balances are non-zero (zero bal lots are only left on their final day)
			if (lotBal.LotSize != 0) || (lotBal.SettledSize != 0) || (lotBal.UnsettledSize != 0) {
				lotBal.LotDt = nextDtStr
				err = s.lotStore.CreateLotBal(ctx, lotBal)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return &v1.RollLotsResponse{Status: "completed"}, nil
}
