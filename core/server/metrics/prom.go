package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type PromRecorder struct {
	decisions    [NumResults]prometheus.Counter
	durationHist prometheus.Histogram
}

func NewPromRecorder(reg prometheus.Registerer) *PromRecorder {
	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ratelimiter",
		Name:      "decisions_total",
		Help:      "Each decision count by its type.",
	}, []string{"result"})

	reqDurHist := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "ratelimiter",
		Name:      "processing_duration",
		Help:      "Execution time of one decision in ms",
		Buckets:   []float64{1, 5, 10, 20, 40, 80, 150, 300, 400},
	})
	reg.MustRegister(cv)
	reg.MustRegister(reqDurHist)
	p := &PromRecorder{
		durationHist: reqDurHist,
	}
	p.decisions[Allowed] = cv.WithLabelValues("allowed")
	p.decisions[Denied] = cv.WithLabelValues("denied")
	p.decisions[Errored] = cv.WithLabelValues("errored")

	return p
}
func (p *PromRecorder) ObserveDecision(r Result, dur time.Duration) {
	p.decisions[r].Inc()
	p.durationHist.Observe(float64(dur.Milliseconds()))
}
