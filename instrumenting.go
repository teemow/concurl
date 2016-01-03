package main

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics"
)

func instrumentingMiddleware(
	requestCount metrics.Counter,
	requestLatency metrics.TimeHistogram,
	countResult metrics.Histogram,
) ServiceMiddleware {
	return func(next ConcurlService) ConcurlService {
		return instrmw{requestCount, requestLatency, countResult, next}
	}
}

type instrmw struct {
	requestCount   metrics.Counter
	requestLatency metrics.TimeHistogram
	countResult    metrics.Histogram
	ConcurlService
}

func (mw instrmw) Get() (output string, err error) {
	defer func(begin time.Time) {
		methodField := metrics.Field{Key: "method", Value: "get"}
		errorField := metrics.Field{Key: "error", Value: fmt.Sprintf("%v", err)}
		mw.requestCount.With(methodField).With(errorField).Add(1)
		mw.requestLatency.With(methodField).With(errorField).Observe(time.Since(begin))
	}(time.Now())

	output, err = mw.ConcurlService.Get()
	return
}
