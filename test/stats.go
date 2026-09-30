package main

import (
	"fmt"
	"math"
	"time"
)

func percentile(d []time.Duration, p float64) time.Duration {
	if len(d) == 0 {
		return 0
	}
	return d[int(float64(len(d)-1)*p)]
}

func peakPerSecond(ts []time.Time) int {
	peak, lo := 0, 0
	for hi := range ts {
		for ts[hi].Sub(ts[lo]) >= time.Second {
			lo++
		}
		if hi-lo+1 >= peak {
			peak = hi - lo + 1
		}
	}
	return peak
}

func Report(stats []*ClientStats, elapsed time.Duration) {
	var allowed, disallowed, errs int
	var allowedLat, limitedLat []time.Duration
	minA, maxA, worstPeak := math.MaxInt, 0, 0

	for _, s := range stats {
		allowed += s.Allowed
		disallowed += s.Disallowed
		errs += s.Errors
		allowedLat = append(allowedLat, s.AllowedLat...)
		limitedLat = append(limitedLat, s.LimitedLat...)
		minA, maxA = min(minA, s.Allowed), max(maxA, s.Allowed)
		worstPeak = max(worstPeak, peakPerSecond(s.AllowedAt))
	}

	total := allowed + disallowed + errs
	fmt.Printf("throughput:      %.0f req/s (%d total, %d transport errors)\n", float64(total)/elapsed.Seconds(), total, errs)
	fmt.Printf("allowed/limited: %d / %d\n", allowed, disallowed)
	fmt.Printf("latency allowed: p50=%v p95=%v p99=%v\n", percentile(allowedLat, .5), percentile(allowedLat, .95), percentile(allowedLat, .99))
	fmt.Printf("latency limited: p50=%v p95=%v p99=%v\n", percentile(limitedLat, .5), percentile(limitedLat, .95), percentile(limitedLat, .99))
	fmt.Printf("peak allowed in 1s window: %d\n", worstPeak)
}
