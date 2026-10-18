package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type PromRecorder struct {
	decisions [NumResults]prometheus.Counter
}

func NewPromRecorder(reg prometheus.Registerer) *PromRecorder {
	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ratelimiter",
		Name:      "decisions_total",
		Help:      "Each decision count by its type.",
	}, []string{"result"})
	reg.MustRegister(cv)

	p := &PromRecorder{}
	p.decisions[Allowed] = cv.WithLabelValues("allowed")
	p.decisions[Denied] = cv.WithLabelValues("denied")
	p.decisions[Errored] = cv.WithLabelValues("errored")

	return p
}
func (p *PromRecorder) ObserveDecision(r Result, _ time.Duration) {
	p.decisions[r].Inc()
}
