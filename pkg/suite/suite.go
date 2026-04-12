package suite

import (
	"context"
	"onlab-bm/pkg/metrics"
	"time"
)

type Suite struct {
	deltas  []*Delta
	fetcher metrics.Fetcher
}

func New(fetcher metrics.Fetcher, deltas ...*Delta) *Suite {
	return &Suite{
		deltas:  deltas,
		fetcher: fetcher,
	}
}

type DeltaReconcileResult struct {
	numGatewayclass int
	numGateways     int
	numHttpRoutes   int
	numServices     int
	reconcileTime   time.Duration
	status          string
	delta           string
}

func (s *Suite) WalkThroughDeltas() []*DeltaReconcileResult {
	ret := make([]*DeltaReconcileResult, 0, len(s.deltas))
	var (
		numGatewayclass = 0
		numGateways     = 0
		numHttpRoutes   = 0
		numServices     = 0
	)
	rchan := make(chan metrics.ReconcileResult, 5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.fetcher.WaitForSuccessfulReconcile(ctx, rchan)
	for _, d := range s.deltas {
		if err := d.Apply(ctx); err != nil {
			ret = append(ret, &DeltaReconcileResult{
				numGatewayclass: numGatewayclass,
				numGateways:     numGateways,
				numHttpRoutes:   numHttpRoutes,
				numServices:     numServices,
				reconcileTime:   time.Duration(0),
				status:          err.Error(),
				delta:           d.String(),
			})
		}
		switch d.UnderlyingType() {
		case DeltaUnderlyingTypeService:
			numServices++
		case DeltaUnderlyingTypeGatewayClass:
			numGatewayclass++
		case DeltaUnderlyingTypeGateway:
			numGateways++
		case DeltaUnderlyingTypeHTTPRoute:
			numHttpRoutes++
		case DeltaUnderlyingTypeUnknown:
		default:
		}
		res := <-rchan
		status := "success"
		if res.Error() != nil {
			status = res.Error().Error()
		}
		ret = append(ret, &DeltaReconcileResult{
			numGatewayclass: numGatewayclass,
			numGateways:     numGateways,
			numHttpRoutes:   numHttpRoutes,
			numServices:     numServices,
			reconcileTime:   res.Delta(),
			status:          status,
			delta:           d.String(),
		})
	}
	return ret
}
