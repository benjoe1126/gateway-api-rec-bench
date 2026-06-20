package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"onlab-bm/pkg/api"
	"onlab-bm/pkg/metrics"
	"onlab-bm/pkg/patch"
	"onlab-bm/pkg/scenario"
	"onlab-bm/pkg/suite"
	"os"
	"strings"
	"sync/atomic"
	"time"

	tracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
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
	testSuite        = flag.String("suite", "default", "suite to run")
	outText          = flag.String("output-text", "results", "text for output")
	outCSV           = flag.String("output-csv", "results.csv", "csv for output")
	baseGatewayCount = flag.Int("base-gateway-count", 1, "number of gateways")
	suiteNameToFunc  = map[string]func(compositeApi *api.CompositeApi) ([]suite.DeltaInterface, error){
		"thousandGatewaysOneGatewayClass": thousandGatewaysOneGatewayClass,
		"semiLarge":                       semiLarge,
		"correctnessCheck":                correctnessCheck,
		"withExponentialIncrease":         withExponentialIncrease,
		"flagBasedBaseConfig":             flagBasedBaseConfig,
	}
)

var (
	globalGatewayClassCount = atomic.Int64{}
	globalGatewayCount      = atomic.Int64{}
	globalHTTPRouteCount    = atomic.Int64{}
	globalSVCCount          = atomic.Int64{}
)
var scTemplate = `
name: test-scenario-{{ .index }}
description: Description for test-scenario-{{ .index }}
resources:
  gateways:
    count: {{ .count }}
deltas: []
initialDelay: {{ .delay }}
`

func main() {
	flag.Parse()
	scg := scenario.ScenarioGenerator{
		ScenarioTemplate: scTemplate,
		Values: map[string]interface{}{
			"index": 0,
			"count": 3,
			"delay": "1s",
		},
	}
	fmt.Println(scg.Generate())
	/*kubeconfig := filepath.Join("tmp", "kubeconfig.yaml")
	kc, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	client, err := dynamic.NewForConfig(kc)
	if err != nil {
		log.Fatal(err)
	}*/

	/*capi := api.NewCompositeApi(client)
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
	time.Sleep(5 * time.Second)
	//traceChan := make(chan *tracev1.ExportTraceServiceRequest, 100)
	//sink := metrics.NewOTELTraceSink(metrics.WithTraceChannel(traceChan))
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(2 * time.Second) // wait for background task to finish running
	}()
	//if err := sink.Start(ctx, 19002); err != nil {
	//	log.Fatalf("failed to start trace sink: %v", err)
	//}
	nchan := make(chan struct{}, 100)
	//go traceConsumePipeline(ctx, traceChan, nchan)
	go watchAndCalculate(ctx, client, fetcher, nchan)
	time.Sleep(2 * time.Second)
	bmSuite.WalkthroughDeltaForTraces(nchan)
	time.Sleep(3 * time.Second)
	cancel()*/
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

func watchAndCalculate(ctx context.Context, cl *dynamic.DynamicClient, fetcher *metrics.Fetcher, notificationChan chan struct{}) {
	reconcileTime := fetcher.MustFetchReconcileTimeSecondsMetric()
	translationTime := fetcher.MustFetchTranslationTimeMetric()
	//currentTransCountMetric := fetcher.MustFetchTranslationCountMetric()
	type times struct {
		translationTime float64
		reconcileTime   float64
	}
	newTimes := make(chan times, 4)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				transTimeNew := fetcher.MustFetchTranslationTimeMetric()
				recTimeNew := fetcher.MustFetchReconcileTimeSecondsMetric()
				if transTimeNew != translationTime {
					newTimes <- times{
						translationTime: transTimeNew,
						reconcileTime:   recTimeNew,
					}
				}
			}
		}
	}()
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
			notificationChan <- struct{}{}
			return
		case nTimes := <-newTimes:
			log.Println("Times are: ", reconcileTime, translationTime)
			newRecTime := nTimes.reconcileTime
			newTransTime := nTimes.translationTime
			log.Println("New times are: ", newRecTime, newTransTime)
			diffRecTime := time.Duration((newRecTime - reconcileTime) * float64(time.Second))
			diffTransTime := time.Duration((newTransTime - translationTime) * float64(time.Second))
			log.Println("Diff times are: ", diffRecTime, diffTransTime)
			reconcileTime = newRecTime
			translationTime = newTransTime
			results = append(results, TraceResult{
				NumGateways:       int(globalGatewayCount.Load()),
				NumHTTPRoutes:     int(globalHTTPRouteCount.Load()),
				NumGatewayClasses: int(globalGatewayClassCount.Load()),
				NumServices:       int(globalSVCCount.Load()),
				ReconcileTime:     diffRecTime,
				TranslationTime:   diffTransTime,
				TotalTime:         diffRecTime + diffTransTime,
			})
			notificationChan <- struct{}{}
		}
	}

}

func (tr *TraceResult) CSV() string {
	return fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d", tr.NumGatewayClasses, tr.NumGateways, tr.NumHTTPRoutes, tr.NumServices, tr.ReconcileTime.Microseconds(), tr.TranslationTime.Microseconds(), tr.TotalTime.Microseconds())
}

func tracesToCSV(traces []TraceResult) string {
	result := ""
	for _, trace := range traces {
		result += trace.CSV() + "\n"
	}
	return result
}

func traceConsumePipeline(ctx context.Context, traceChan chan *tracev1.ExportTraceServiceRequest, notiChan chan<- struct{}) {
	log.Println("starting trace consume pipeline")
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
			return

		case trace := <-traceChan:
			recTime := time.Duration(0)
			transTime := time.Duration(0)
			totalTime := time.Duration(0)
			for _, rspan := range trace.ResourceSpans {
				for _, sspan := range rspan.ScopeSpans {
					for _, span := range sspan.Spans {
						if strings.HasPrefix(span.Name, "GatewayAPIReconciler.Reconcile") {
							recTime = time.Duration(span.GetEndTimeUnixNano() - span.GetStartTimeUnixNano())
							log.Printf("span: %s took %v to reconcile", span.Name, recTime)
						} else if strings.Contains(span.Name, "GatewayApiRunner.subscribeAndTranslate") {
							transTime = time.Duration(span.GetEndTimeUnixNano() - span.GetStartTimeUnixNano())
							log.Printf("span: %s took %v to translate", span.Name, transTime)
						}
					}
					totalTime = transTime + recTime
				}
			}
			log.Println(globalGatewayClassCount.Load(), globalGatewayCount.Load(), globalHTTPRouteCount.Load(), globalSVCCount.Load())
			results = append(results, TraceResult{
				NumGateways:       int(globalGatewayCount.Load()),
				NumHTTPRoutes:     int(globalHTTPRouteCount.Load()),
				NumGatewayClasses: int(globalGatewayClassCount.Load()),
				NumServices:       int(globalSVCCount.Load()),
				ReconcileTime:     recTime,
				TranslationTime:   transTime,
				TotalTime:         totalTime,
			})
			log.Println("finished consumption for now, sending message to notification pipeline")
			notiChan <- struct{}{}
		}
	}
}

var resourceBatchCounter = atomic.Int64{}

func bachCounterIncHook(at *atomic.Int64) func(...any) {
	return func(_ ...any) {
		at.Add(1)
	}
}

func modHook(at *atomic.Int64, diff int64) func(...any) {
	return func(a ...any) {
		at.Add(diff)
	}
}

func flagBasedBaseConfig(capi api.CompositeApi) ([]suite.DeltaInterface, error) {
	if err := capi.GatewayClass().Create(context.Background(), &baseGatewayClass); err != nil {
		return nil, err
	}
	globalGatewayClassCount.Add(1)
	if err := capi.Service().Create(context.Background(), &baseService); err != nil {
		log.Println(err)
		return nil, err
	}
	for i := range *baseGatewayCount {
		globalGatewayCount.Add(1)
		gw := baseGateway.DeepCopy()
		gw.Name = fmt.Sprintf(baseGateway.Name, i)
		if err := capi.Gateway().Create(context.Background(), gw); err != nil {
			return nil, err
		}
	}
	gw := baseGateway.DeepCopy()
	gw.Name = fmt.Sprintf(baseGateway.Name, globalGatewayCount.Load()+5)
	deltas := make([]suite.DeltaInterface, 0, 4)
	deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.Gateway(), gw, suite.WithHooks(modHook(&globalGatewayCount, 1))))
	deltas = append(deltas, suite.NewDelta(suite.DeltaOpDelete, capi.Gateway(), gw, suite.WithHooks(modHook(&globalGatewayCount, -1))))
	deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.Gateway(), gw, suite.WithHooks(modHook(&globalGatewayCount, 1))))
	patches := []patch.JsonPatch{
		{
			Op:    "add",
			Path:  "/spec/listeners/0/port",
			Value: 8080,
		},
	}
	deltas = append(deltas, suite.NewDelta(suite.DeltaOpModify, capi.Gateway(), gw, suite.WithPatches(patches...)))
	return deltas, nil

}

func thousandGatewaysOneGatewayClass(capi *api.CompositeApi) ([]suite.DeltaInterface, error) {
	if err := capi.GatewayClass().Create(context.Background(), &baseGatewayClass); err != nil {
		return nil, err
	}
	globalGatewayClassCount.Add(1)
	deltas := make([]suite.DeltaInterface, 0, 8000)
	//first we add 1000 gateways, delete each, readd them, then modify it slightly
	for i := range 1000 {
		gw := baseGateway.DeepCopy()
		gw.Name = fmt.Sprintf(baseGateway.Name, i)
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.Gateway(), gw, suite.WithHooks(modHook(&globalGatewayCount, 1))))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpDelete, capi.Gateway(), gw, suite.WithHooks(modHook(&globalGatewayCount, -1))))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.Gateway(), gw, suite.WithHooks(modHook(&globalGatewayCount, 1))))
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

func withExponentialIncrease(capi *api.CompositeApi) ([]suite.DeltaInterface, error) {
	//base is 1 gatewayclass, 1 gateway, and 1 route
	// adding in 2^i gateways with each delta
	if err := capi.GatewayClass().Create(context.Background(), &baseGatewayClass); err != nil {
		return nil, err
	}
	globalGatewayClassCount.Add(1)
	if err := capi.Service().Create(context.Background(), &baseService); err != nil {
		log.Println(err)
		return nil, err
	}
	globalSVCCount.Add(1)
	globalGatewayCount.Add(1)
	gwTmp := baseGateway.DeepCopy()
	gwTmp.Name = fmt.Sprintf(baseGateway.Name, 0)
	if err := capi.Gateway().Create(context.Background(), gwTmp); err != nil {
		return nil, err
	}
	deltas := make([]suite.DeltaInterface, 0, 10)
	gwCount := 1
	for i := range 10 {
		deltasTmp := make([]suite.DeltaInterface, 0, 10)
		for range int64(math.Pow(2, float64(i))) {
			gw := baseGateway.DeepCopy()
			gw.Name = fmt.Sprintf(baseGateway.Name, gwCount)
			gwCount++
			deltasTmp = append(deltasTmp, suite.NewDelta(suite.DeltaOpAdd, capi.Gateway(), gw, suite.WithHooks(bachCounterIncHook(&resourceBatchCounter), modHook(&globalGatewayCount, 1))))
		}
		cDelta := suite.NewCompositeDelta(deltasTmp...)
		deltas = append(deltas, cDelta)
	}
	return deltas, nil
}

// create 1 gwclass, 1000 gateways and 10 routes/gateway
func semiLarge(capi *api.CompositeApi) ([]suite.DeltaInterface, error) {
	if err := capi.GatewayClass().Create(context.Background(), &baseGatewayClass); err != nil {
		return nil, err
	}
	if err := capi.Service().Create(context.Background(), &baseService); err != nil {
		log.Println(err)
		return nil, err
	}
	for i := range 100 {
		globalGatewayCount.Add(1)
		gw := baseGateway.DeepCopy()
		gw.Name = fmt.Sprintf(baseGateway.Name, i)
		if err := capi.Gateway().Create(context.Background(), gw); err != nil {
			return nil, err
		}
	}
	time.Sleep(2 * time.Second)
	for i := range 1000 {
		globalHTTPRouteCount.Add(1)
		route := baseHttpRoute.DeepCopy()
		route.Name = fmt.Sprintf(baseHttpRoute.Name, i)
		route.Spec.Rules[0].BackendRefs[0].Name = gatewayv1.ObjectName(fmt.Sprintf(baseGateway.Name, i%1000))
		if err := capi.HTTPRoute().Create(context.Background(), route); err != nil {
			return nil, err
		}
	}
	deltas := make([]suite.DeltaInterface, 0, 100)
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
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.HTTPRoute(), route, suite.WithHooks(modHook(&globalHTTPRouteCount, 1))))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpDelete, capi.HTTPRoute(), route, suite.WithHooks(modHook(&globalHTTPRouteCount, -1))))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.HTTPRoute(), route, suite.WithHooks(modHook(&globalHTTPRouteCount, 1))))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpModify, capi.HTTPRoute(), route, suite.WithPatches(patches...)))
	}
	return deltas, nil

}

func correctnessCheck(capi *api.CompositeApi) ([]suite.DeltaInterface, error) {
	if err := capi.GatewayClass().Create(context.Background(), &baseGatewayClass); err != nil {
		return nil, err
	}
	globalGatewayClassCount.Add(1)
	if err := capi.Service().Create(context.Background(), &baseService); err != nil {
		return nil, err
	}
	globalSVCCount.Add(1)
	deltas := make([]suite.DeltaInterface, 0, 10)
	for i := range 10 {
		gw := baseGateway.DeepCopy()
		gw.Name = fmt.Sprintf(baseGateway.Name, i)
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.Gateway(), gw, suite.WithHooks(modHook(&globalGatewayCount, 1))))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpDelete, capi.Gateway(), gw, suite.WithHooks(modHook(&globalGatewayCount, -1))))
		deltas = append(deltas, suite.NewDelta(suite.DeltaOpAdd, capi.Gateway(), gw, suite.WithHooks(modHook(&globalGatewayCount, 1))))
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
