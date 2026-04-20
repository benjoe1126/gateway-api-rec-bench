package metrics

import (
	"context"
	"log"

	v1 "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	v1trace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

type OTELSink struct {
	Ot *V1OtelSink
	v1trace.TraceServiceServer
}

type V1OtelSink struct {
	v1.MetricsServiceServer
}

func (ot *V1OtelSink) Export(ctx context.Context, req *v1.ExportMetricsServiceRequest) (*v1.ExportMetricsServiceResponse, error) {
	for _, rs := range req.ResourceMetrics {
		log.Println(rs.Resource)
	}
	return &v1.ExportMetricsServiceResponse{}, nil
}

func (s *OTELSink) Export(ctx context.Context, req *v1trace.ExportTraceServiceRequest) (*v1trace.ExportTraceServiceResponse, error) {
	for _, s := range req.ResourceSpans {
		logResourceSpans(s)
	}
	return &v1trace.ExportTraceServiceResponse{}, nil
}

func logResourceSpans(rs *tracepb.ResourceSpans) {
	for _, ils := range rs.ScopeSpans {
		for _, span := range ils.Spans {
			log.Printf(
				"trace_id=%x span_id=%x name=%s start=%d end=%d",
				span.TraceId,
				span.SpanId,
				span.Name,
				span.StartTimeUnixNano,
				span.EndTimeUnixNano,
			)
		}
	}
}
