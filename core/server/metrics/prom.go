package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type PromRecorder struct {
	decisions    [NumResults]prometheus.Counter
	durationHist [NumResults]prometheus.Observer
}

func NewPromRecorder(reg prometheus.Registerer) *PromRecorder {
	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ratelimiter",
		Name:      "decisions_total",
		Help:      "Each decision count by its type.",
	}, []string{"result"})

	reqDurHistVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "ratelimiter",
		Name:      "decision_duration_seconds",
		Help:      "Execution time of one decision in seconds.",
		Buckets:   []float64{0.00005, 0.0001, 0.0005, 0.001, 0.1, 0.5, 1, 2},
	}, []string{"result"})
	reg.MustRegister(counterVec)
	reg.MustRegister(reqDurHistVec)
	p := &PromRecorder{}

	p.decisions[Allowed] = counterVec.WithLabelValues("allowed")
	p.decisions[Denied] = counterVec.WithLabelValues("denied")
	p.decisions[Errored] = counterVec.WithLabelValues("errored")
	p.durationHist[Allowed] = reqDurHistVec.WithLabelValues("allowed")
	p.durationHist[Denied] = reqDurHistVec.WithLabelValues("denied")
	p.durationHist[Errored] = reqDurHistVec.WithLabelValues("errored")

	return p
}
func (p *PromRecorder) ObserveDecision(r Result, dur time.Duration) {
	p.decisions[r].Inc()
	p.durationHist[r].Observe(dur.Seconds())
}
