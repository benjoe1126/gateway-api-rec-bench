package metrics

import "time"

type ReconcileResult struct {
	delta time.Duration
	err   error
}

func (r *ReconcileResult) Delta() time.Duration {
	return r.delta
}

func (r *ReconcileResult) Error() error {
	return r.err
}
