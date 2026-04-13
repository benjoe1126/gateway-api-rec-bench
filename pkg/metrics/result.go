package metrics

type ReconcileResult struct {
	delta float64
	err   error
}

func (r *ReconcileResult) Delta() float64 {
	return r.delta
}

func (r *ReconcileResult) Error() error {
	return r.err
}
