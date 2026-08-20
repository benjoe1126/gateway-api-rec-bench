package metrics

import "time"

/*
Record represents a single datapoint that contains the time it took to reconcile and/or translate k8s resource to
actual configuration, as well as the number of configs
*/
type Record struct {
	ReconcileTime        time.Duration `json:"reconcile_time" yaml:"reconcile_time"`
	TranslationTime      time.Duration `json:"translation_time" yaml:"translation_time"`
	TotalDeltaTime       time.Duration `json:"total_delta_time" yaml:"total_delta_time"`
	ResourceAddedCount   int           `json:"resource_added_count" yaml:"resource_added_count"`
	ResourceRemovedCount int           `json:"resource_removed_count" yaml:"resource_removed_count"`
	ResourceUpdatedCount int           `json:"resource_updated_count" yaml:"resource_updated_count"`
}
