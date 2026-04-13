package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

const (
	reconcileTimeMetric        = "controller_runtime_reconcile_time_seconds"
	successfulReconcileMetrics = "controller_runtime_reconcile_total"
)

type Fetcher struct {
	metricsURL string
}

func NewFetcher(metricsURL string) *Fetcher {
	return &Fetcher{
		metricsURL: metricsURL,
	}
}

var WaitInitiateError = fmt.Errorf("wait initiate")

func (f *Fetcher) WaitForSuccessfulReconcile(ctx context.Context, ch chan<- ReconcileResult) {
	initialReconciles, err := f.FetchSuccessfulReconcileMetric()
	if err != nil {
		log.Printf("failed to fetch initial reconciles: %s", err)
		ch <- ReconcileResult{err: err}
		return
	}
	initialReconcileTime := f.MustFetchReconcileTimeSecondsMetric()
	ch <- ReconcileResult{err: WaitInitiateError}
	for {
		select {
		case <-ctx.Done():
			ch <- ReconcileResult{err: ctx.Err()}
			log.Println(ctx.Err())
			return
		default:
			newReconciles, err := f.FetchSuccessfulReconcileMetric()
			if err != nil {
				log.Println(err)
				ch <- ReconcileResult{err: err}
				return
			}
			if newReconciles != initialReconciles {
				nrec := f.MustFetchReconcileTimeSecondsMetric()
				delta := nrec - initialReconcileTime
				initialReconciles = newReconciles
				initialReconcileTime = nrec
				ch <- ReconcileResult{delta: delta}
			}
		}
	}
}

func (f *Fetcher) MustFetchReconcileTimeSecondsMetric() float64 {
	ret, err := f.FetchReconcileTimeMetric()
	if err != nil {
		panic(err)
	}
	return ret
}

func (f *Fetcher) FetchReconcileTimeMetric() (float64, error) {
	metric, err := fetchAndFilterMetric(f.metricsURL, reconcileTimeMetric)
	if err != nil {
		return 0, err
	}
	return getMetricValue(metric.Metric[0]), nil
}

func (f *Fetcher) MustFetchSuccessfulReconcileMetric() int {
	ret, err := f.FetchSuccessfulReconcileMetric()
	if err != nil {
		panic(err)
	}
	return ret
}

func (f *Fetcher) FetchSuccessfulReconcileMetric() (int, error) {
	metrics, err := fetchAndFilterMetric(f.metricsURL, successfulReconcileMetrics)
	if err != nil {
		return 0.0, fmt.Errorf("error fetching metrics: %v", err)
	}
	for _, metric := range metrics.Metric {
		for _, label := range metric.Label {
			if *label.Name == "result" && *label.Value == "success" {
				return int(metric.Counter.GetValue()), nil
			}
		}
	}
	return 0.0, fmt.Errorf("metric not found")
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

func floatToSecond(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
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
