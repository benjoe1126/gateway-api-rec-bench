package suite

import (
	"context"
	"fmt"
	"log"
	"onlab-bm/pkg/metrics"
)

type Suite struct {
	deltas  []*Delta
	fetcher *metrics.Fetcher
}

func New(fetcher *metrics.Fetcher, deltas ...*Delta) *Suite {
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
	reconcileTime   float64
	status          string
	delta           string
}

func (d *DeltaReconcileResult) CSV() string {
	return fmt.Sprintf("%d,%d,%d,%d,%v", d.numGatewayclass, d.numGateways, d.numHttpRoutes, d.numServices, d.reconcileTime)
}

func (d *DeltaReconcileResult) String() string {
	resultString := `
GatewayClasses: %d
Gateways: %d
HttpRoutes: %d
Services: %d
Reconcile time: %v
Status: %s
delta: %s
----------------------------------------------
`
	return fmt.Sprintf(resultString, d.numGatewayclass, d.numGateways, d.numHttpRoutes, d.numServices, d.reconcileTime, d.status, d.delta)
}

func (s *Suite) WalkThroughDeltas() []*DeltaReconcileResult {
	ret := make([]*DeltaReconcileResult, 0, len(s.deltas))
	var (
		numGatewayclass = 1
		numGateways     = 0
		numHttpRoutes   = 0
		numServices     = 0
	)
	rchan := make(chan metrics.ReconcileResult, 5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.fetcher.WaitForSuccessfulReconcile(ctx, rchan)
	<-rchan
	for _, d := range s.deltas {
		log.Println("Applying delta ", d.String())
		if err := d.Apply(ctx); err != nil {
			log.Println("Error applying delta ", d.String(), err)
			ret = append(ret, &DeltaReconcileResult{
				numGatewayclass: numGatewayclass,
				numGateways:     numGateways,
				numHttpRoutes:   numHttpRoutes,
				numServices:     numServices,
				reconcileTime:   0.0,
				status:          err.Error(),
				delta:           d.String(),
			})
			continue
		}
		adder := 0
		if d.op == DeltaOpAdd {
			adder++
		} else if d.op == DeltaOpDelete {
			adder--
		}
		switch d.UnderlyingType() {
		case DeltaUnderlyingTypeService:
			numServices += adder
		case DeltaUnderlyingTypeGatewayClass:
			numGatewayclass += adder
		case DeltaUnderlyingTypeGateway:
			numGateways += adder
		case DeltaUnderlyingTypeHTTPRoute:
			numHttpRoutes += adder
		case DeltaUnderlyingTypeUnknown:
		default:
		}
		res := <-rchan
		log.Println("Finished applying delta ", d.String())
		log.Println("Applying delta took", res.Delta(), " seconds")
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
