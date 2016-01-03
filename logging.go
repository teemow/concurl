package main

import (
	"time"

	"github.com/go-kit/kit/log"
)

func loggingMiddleware(logger log.Logger) ServiceMiddleware {
	return func(next ConcurlService) ConcurlService {
		return logmw{logger, next}
	}
}

type logmw struct {
	logger log.Logger
	ConcurlService
}

func (mw logmw) Get() (output string, err error) {
	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "get",
			"output", output,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	output, err = mw.ConcurlService.Get()
	return
}
