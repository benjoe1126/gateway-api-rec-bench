from http.server import BaseHTTPRequestHandler, HTTPServer
import gzip

try:
    from opentelemetry.proto.collector.metrics.v1.metrics_service_pb2 import (
        ExportMetricsServiceRequest,
    )
except ImportError:
    raise SystemExit(
        "Install dependencies:\n"
        "pip install opentelemetry-proto protobuf"
    )


class OTELMetricsHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path != "/v1/metrics":
            self.send_response(404)
            self.end_headers()
            return

        content_length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(content_length)

        if self.headers.get("Content-Encoding") == "gzip":
            body = gzip.decompress(body)

        req = ExportMetricsServiceRequest()
        req.ParseFromString(body)

        for resource_metric in req.resource_metrics:
            resource_attrs = {
                attr.key: (
                    attr.value.string_value
                    or attr.value.int_value
                    or attr.value.double_value
                    or attr.value.bool_value
                )
                for attr in resource_metric.resource.attributes
            }

            print("\n=== Resource ===")
            print(resource_attrs)

            for scope_metric in resource_metric.scope_metrics:
                for metric in scope_metric.metrics:
                    print(f"\nMetric: {metric.name}")
                    print(f"Description: {metric.description}")
                    print(f"Unit: {metric.unit}")

                    if metric.HasField("gauge"):
                        for dp in metric.gauge.data_points:
                            print_data_point(dp)

                    elif metric.HasField("sum"):
                        for dp in metric.sum.data_points:
                            print_data_point(dp)

                    elif metric.HasField("histogram"):
                        for dp in metric.histogram.data_points:
                            print_histogram(dp)

        self.send_response(200)
        self.end_headers()

    def log_message(self, format, *args):
        return


def attrs_to_dict(attrs):
    out = {}
    for attr in attrs:
        v = attr.value
        out[attr.key] = (
            v.string_value
            or v.int_value
            or v.double_value
            or v.bool_value
        )
    return out


def print_data_point(dp):
    value = None

    if hasattr(dp, "as_int") and dp.as_int:
        value = dp.as_int
    elif hasattr(dp, "as_double") and dp.as_double:
        value = dp.as_double

    print(
        {
            "attributes": attrs_to_dict(dp.attributes),
            "start_time_unix_nano": dp.start_time_unix_nano,
            "time_unix_nano": dp.time_unix_nano,
            "value": value,
        }
    )


def print_histogram(dp):
    print(
        {
            "attributes": attrs_to_dict(dp.attributes),
            "count": dp.count,
            "sum": dp.sum,
            "bucket_counts": list(dp.bucket_counts),
            "explicit_bounds": list(dp.explicit_bounds),
        }
    )


if __name__ == "__main__":
    server = HTTPServer(("0.0.0.0", 19003), OTELMetricsHandler)
    print("OTEL metrics sink listening on http://0.0.0.0:19003/v1/metrics")
    server.serve_forever()