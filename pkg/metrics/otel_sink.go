package metrics

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	v1trace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc"
)

type OTELTraceSink struct {
	v1trace.TraceServiceServer
	traceChan chan *v1trace.ExportTraceServiceRequest
}

type OtelTraceOpt func(*OTELTraceSink)

func WithTraceChannel(ch chan *v1trace.ExportTraceServiceRequest) OtelTraceOpt {
	return func(o *OTELTraceSink) {
		o.traceChan = ch
	}
}

func NewOTELTraceSink(opts ...OtelTraceOpt) *OTELTraceSink {
	ret := &OTELTraceSink{
		traceChan: make(chan *v1trace.ExportTraceServiceRequest),
	}
	for _, o := range opts {
		o(ret)
	}
	return ret
}

func (s *OTELTraceSink) Start(ctx context.Context, port uint16) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	v1trace.RegisterTraceServiceServer(grpcServer, s)
	errchan := make(chan error)
	go func() {
		log.Println("OTLP gRPC trace sink listening on :19002")
		errchan <- grpcServer.Serve(lis)
		log.Println("OTLP gRPC trace sink stopped")
	}()
	go func() {
		select {
		case <-ctx.Done():
			grpcServer.GracefulStop()
		case err := <-errchan:
			log.Fatalf("OTLP gRPC trace sink error: %v", err)
		}
	}()
	return nil
}

func (s *OTELTraceSink) Export(ctx context.Context, req *v1trace.ExportTraceServiceRequest) (*v1trace.ExportTraceServiceResponse, error) {
	/*	for _, r := range req.ResourceSpans {
		logResourceSpans(r)
	}*/
	s.traceChan <- req
	return &v1trace.ExportTraceServiceResponse{}, nil
}

func logResourceSpans(rs *tracepb.ResourceSpans) {
	for _, ils := range rs.ScopeSpans {
		for _, span := range ils.Spans {
			log.Printf(
				"trace_id=%x span_id=%x name=%s duration=%s",
				span.TraceId,
				span.SpanId,
				span.Name,
				(time.Nanosecond * (time.Duration(span.EndTimeUnixNano) - time.Duration(span.StartTimeUnixNano))).String(),
			)
		}
	}
}
