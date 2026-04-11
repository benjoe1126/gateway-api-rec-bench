package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"onlab-bm/pkg/api"
	"path/filepath"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	reconcileTimeMetric = "controller_runtime_reconcile_time_seconds"
	metricsUrl          = "http://localhost:19001/metrics"
)

var (
	gatewayClassDescription = "Base gateway class for measurements"
	httPort                 = gatewayv1.PortNumber(80)
	baseGatewayClass        = gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "base-gatewayclass",
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayv1.GroupVersion.String(),
			Kind:       "GatewayClass",
		},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: "gateway.envoyproxy.io/gatewayclass-controller",
			Description:    &gatewayClassDescription,
		},
	}
	baseGateway = gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "base-gateway-%d",
			Namespace: metav1.NamespaceDefault,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayv1.GroupVersion.String(),
			Kind:       "Gateway",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(baseGatewayClass.Name),
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Port:     80,
					Protocol: gatewayv1.HTTPProtocolType,
				},
			},
		},
	}
	baseService = v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "http-backend",
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
	baseHttpRoute = gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name: "base-http-route-%d",
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayv1.GroupVersion.String(),
			Kind:       "HTTPRoute",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: gatewayv1.ObjectName(baseService.Name),
									Port: &httPort,
								},
							},
						},
					},
				},
			},
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name: "base-gateway-%d",
					},
				},
			},
		},
	}
)

func waitForSuccessfulReconcile(ctx context.Context, notification chan<- error) {
	initialReconciles, err := fetchSuccesfulReconcileTotal()
	if err != nil {
		notification <- err
		return
	}
	for {
		select {
		case <-ctx.Done():
			notification <- ctx.Err()
			return
		default:
			newReconciles, err := fetchSuccesfulReconcileTotal()
			if err != nil {
				notification <- err
				return
			}
			if newReconciles != initialReconciles {
				notification <- nil
				return
			}
		}
	}
}

var (
	gatewayCounter = 0
	routeCounter   = 0
)

func main() {
	notification := make(chan error)
	defer close(notification)
	kubeconfig := filepath.Join("tmp", "kubeconfig.yaml")
	kc, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	client, err := dynamic.NewForConfig(kc)
	if err != nil {
		log.Fatal(err)
	}
	capi := api.NewCompositeApi(client)
	gg := baseGateway.DeepCopy()
	gg.Name = "something-something"
	if err := capi.Gateway().Create(context.Background(), gg); err != nil {
		log.Fatal(err)
	}
}

func floatToSecond(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}

func mustFetchReconcileTimeSum() time.Duration {
	ret, err := fetchReconcileTimeSum()
	if err != nil {
		log.Fatal(err)
	}
	return ret
}

func mustFetchSuccesfulReconcileTotal() float64 {
	ret, err := fetchSuccesfulReconcileTotal()
	if err != nil {
		log.Fatal(err)
	}
	return ret
}

func fetchReconcileTimeSum() (time.Duration, error) {
	metric, err := fetchAndFilterMetric(metricsUrl, reconcileTimeMetric)
	if err != nil {
		return 0, err
	}
	return floatToSecond(getMetricValue(metric.Metric[0])), nil

}

func fetchSuccesfulReconcileTotal() (float64, error) {
	metrics, err := fetchAndFilterMetric(metricsUrl, "controller_runtime_reconcile_total")
	if err != nil {
		log.Printf("error fetching metrics: %v", err)
		return 0, err
	}
	for _, metric := range metrics.Metric {
		for _, label := range metric.Label {
			if *label.Name == "result" && *label.Value == "success" {
				return metric.Counter.GetValue(), nil
			}
		}
	}
	return 0, nil
}

func fetchAndFilterMetric(url, metricName string) (*dto.MetricFamily, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	parser := expfmt.NewTextParser(model.UTF8Validation)
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, err
	}
	if mf, ok := metricFamilies[metricName]; ok {
		return mf, nil
	}

	return nil, fmt.Errorf("metric not found: %s", metricName)
}

func getMetricValue(m *dto.Metric) float64 {
	if m.Gauge != nil {
		return m.Gauge.GetValue()
	}
	if m.Counter != nil {
		return m.Counter.GetValue()
	}
	if m.Untyped != nil {
		return m.Untyped.GetValue()
	}
	if m.Summary != nil && len(m.Summary.Quantile) > 0 {
		return m.Summary.Quantile[0].GetValue()
	}
	if m.Histogram != nil {
		return float64(m.Histogram.GetSampleSum())
	}
	return 0
}
