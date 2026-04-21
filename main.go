package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"onlab-bm/pkg/api"
	"onlab-bm/pkg/metrics"
	"onlab-bm/pkg/patch"
	"onlab-bm/pkg/suite"
	"os"
	"strings"
	"time"

	tracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	metricsUrl = "http://localhost:19001/metrics"
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
			Name:      "http-backend",
			Namespace: metav1.NamespaceDefault,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Ports: []v1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
	serviceGroup  = gatewayv1.Group(baseService.GroupVersionKind().Group)
	serviceKind   = gatewayv1.Kind(baseService.GroupVersionKind().Kind)
	serviceNs     = gatewayv1.Namespace(baseService.GetNamespace())
	baseHttpRoute = gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "base-http-route-%d",
			Namespace: metav1.NamespaceDefault,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayv1.GroupVersion.String(),
			Kind:       "HTTPRoute",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{},
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
						Name: "base-gateway-0",
					},
				},
			},
		},
	}
)

var (
	testSuite       = flag.String("suite", "default", "suite to run")
	outText         = flag.String("output-text", "results", "text for output")
	outCSV          = flag.String("output-csv", "results.csv", "csv for output")
	suiteNameToFunc = map[string]func(compositeApi *api.CompositeApi) ([]*suite.Delta, error){
		"thousandGatewaysOneGatewayClass": thousandGatewaysOneGatewayClass,
		"semiLarge":                       semiLarge,
	}
)

var (
	globalGatewayClassCount = 0
	globalGatewayCount      = 0
	globalHTTPRouteCount    = 0
	globalSVCCount          = 0
)

func main() {
	/*flag.Parse()
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
	/*routes, err := capi.HTTPRoute().List(context.Background(), "default")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("existing routes: ", *routes[0].(*gatewayv1.HTTPRoute))
	fetcher := metrics.NewFetcher(metricsUrl)
	f := suiteNameToFunc[*testSuite]
	if f == nil {
		log.Fatalf("unknown test suite: %s", *testSuite)
	}
	deltas, err := f(capi)
	if err != nil {
		log.Fatal(err)
	}
	bmSuite := suite.New(fetcher, deltas...)
	// wait a bit for initial reconciles
	time.Sleep(4 * time.Second)
	results := bmSuite.WalkThroughDeltas()
	of, err := os.Create(*outText)
	if err != nil {
		log.Fatal(err)
	}
	defer of.Close()
	csv, err := os.Create(*outCSV)
	if err != nil {
		log.Fatal(err)
	}
	defer csv.Close()
	for _, result := range results {
		of.WriteString(result.String())
		of.WriteString("\n")
		csv.WriteString(result.CSV())
		csv.WriteString("\n")
	}*/
	lis, err := net.Listen("tcp", fmt.Sprintf(":19002"))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	traceChan := make(chan *tracev1.ExportTraceServiceRequest, 100)
	sink := metrics.NewOTELTraceSink(metrics.WithTraceChannel(traceChan))
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(5 * time.Second) // wait for background task to finish running
	}()
	if err := sink.Start(ctx, 19002); err != nil {
		log.Fatalf("failed to start trace sink: %v", err)
	}
	go traceConsumePipeline(ctx, traceChan)
	log.Println("OTLP gRPC trace sink listening on :19002")
	if err := grpcServer.Serve(lis); err != nil {
		cancel()
		log.Fatalf("failed to serve: %v", err)
	}
}

type TraceResult struct {
	NumGateways       int           `yaml:"numGateways"`
	NumHTTPRoutes     int           `yaml:"numHTTPRoutes"`
	NumGatewayClasses int           `yaml:"numGatewayClasses"`
	NumServices       int           `yaml:"numServices"`
	ReconcileTime     time.Duration `yaml:"reconcileTime"`
	TranslationTime   time.Duration `yaml:"translationTime"`
	TotalTime         time.Duration `yaml:"totalTime"`
}

func (tr *TraceResult) CSV() string {
	return fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d", tr.NumGatewayClasses, tr.NumGateways, tr.NumHTTPRoutes, tr.NumServices, tr.ReconcileTime.Milliseconds(), tr.TranslationTime.Milliseconds(), tr.TotalTime.Milliseconds())
}

func tracesToCSV(traces []TraceResult) string {
	result := ""
	for _, trace := range traces {
		result += trace.CSV() + "\n"
	}
	return result
}

func traceConsumePipeline(ctx context.Context, traceChan chan *tracev1.ExportTraceServiceRequest) {

	results := make([]TraceResult, 0, 1000)
	for {
		select {
		case <-ctx.Done():
			of, err := os.Create(*outText)
			if err != nil {
				log.Fatalf("failed to create output file: %v", err)
			}
			defer of.Close()
			ocsv, err := os.Create(*outCSV)
			if err != nil {
				log.Fatalf("failed to create output file: %v", err)
			}
			defer ocsv.Close()
			outBytes, _ := yaml.Marshal(results)
			of.Write(outBytes)
			ocsv.WriteString(tracesToCSV(results))

		case trace := <-traceChan:
			recTime := time.Duration(0)
			transTime := time.Duration(0)
			totalTime := time.Duration(0)
			for _, rspan := range trace.ResourceSpans {
				for _, sspan := range rspan.ScopeSpans {
					for _, span := range sspan.Spans {
						if strings.HasPrefix(span.Name, "GatewayAPIReconciler.Reconcile") {
							recTime = time.Nanosecond * time.Duration(span.GetEndTimeUnixNano()-span.GetStartTimeUnixNano())
						} else if strings.HasPrefix(span.Name, "GatewayAPIReconciler.Reconcile") {
							transTime = time.Nanosecond * time.Duration(span.GetEndTimeUnixNano()-span.GetStartTimeUnixNano())
						}
					}
					totalTime = transTime + recTime
					results = append(results, TraceResult{
						NumGateways:       globalGatewayClassCount,
						NumHTTPRoutes:     globalHTTPRouteCount,
						NumGatewayClasses: globalGatewayClassCount,
						NumServices:       globalSVCCount,
						ReconcileTime:     recTime,
						TranslationTime:   transTime,
						TotalTime:         totalTime,
					})
				}
			}
		}
	}
}

func thousandGatewaysOneGatewayClass(capi *api.CompositeApi) ([]*suite.Delta, error) {
	if err := capi.GatewayClass().Create(context.Background(), &baseGatewayClass); err != nil {
		return nil, err
	}
	globalGatewayClassCount++
	deltas := make([]*suite.Delta, 0, 8000)
	//first we add 1000 gateways, delete each, readd them, then modify it slightly
	for i := range 1000 {
		gw := baseGateway.DeepCopy()
		gw.Name = fmt.Sprintf(baseGateway.Name, i)
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.Gateway(), gw, suite.WithHooks(func(...any) {
			globalGatewayCount++
		})))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpDelete, capi.Gateway(), gw, suite.WithHooks(func(...any) {
			globalGatewayCount--
		})))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.Gateway(), gw, suite.WithHooks(func(...any) {
			globalGatewayCount++
		})))
		patches := []patch.JsonPatch{
			{
				Op:    "add",
				Path:  "/spec/listeners/0/port",
				Value: 8080,
			},
		}
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpModify, capi.Gateway(), gw, suite.WithPatches(patches...)))
	}
	return deltas, nil
}

// create 1 gwclass, 1000 gateways and 10 routes/gateway
func semiLarge(capi *api.CompositeApi) ([]*suite.Delta, error) {
	if err := capi.GatewayClass().Create(context.Background(), &baseGatewayClass); err != nil {
		return nil, err
	}
	if err := capi.Service().Create(context.Background(), &baseService); err != nil {
		log.Println(err)
		return nil, err
	}
	for i := range 100 {
		globalGatewayCount++
		gw := baseGateway.DeepCopy()
		gw.Name = fmt.Sprintf(baseGateway.Name, i)
		if err := capi.Gateway().Create(context.Background(), gw); err != nil {
			return nil, err
		}
	}
	time.Sleep(2 * time.Second)
	for i := range 1000 {
		globalHTTPRouteCount++
		route := baseHttpRoute.DeepCopy()
		route.Name = fmt.Sprintf(baseHttpRoute.Name, i)
		route.Spec.Rules[0].BackendRefs[0].Name = gatewayv1.ObjectName(fmt.Sprintf(baseGateway.Name, i%1000))
		if err := capi.HTTPRoute().Create(context.Background(), route); err != nil {
			return nil, err
		}
	}
	deltas := make([]*suite.Delta, 0, 100)
	for i := range 100 {
		route := baseHttpRoute.DeepCopy()
		route.Name = fmt.Sprintf(baseHttpRoute.Name, i+10000)
		patches := []patch.JsonPatch{
			{
				Op:   patch.PatchOpAdd,
				Path: "/spec/hostnames",
				Value: []string{
					"new.example.com",
				},
			},
		}
		route.Spec.ParentRefs[0].Name = gatewayv1.ObjectName(fmt.Sprintf(baseGateway.Name, i))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.HTTPRoute(), route, suite.WithHooks(func(...any) {
			globalHTTPRouteCount++
		})))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpDelete, capi.HTTPRoute(), route, suite.WithHooks(func(...any) {
			globalHTTPRouteCount--
		})))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.HTTPRoute(), route, suite.WithHooks(func(...any) {
			globalHTTPRouteCount++
		})))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpModify, capi.HTTPRoute(), route, suite.WithPatches(patches...)))
	}
	return deltas, nil

}
