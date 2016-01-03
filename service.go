package main

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	jujuratelimit "github.com/juju/ratelimit"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/loadbalancer"
	"github.com/go-kit/kit/loadbalancer/static"
	"github.com/go-kit/kit/log"
	kitratelimit "github.com/go-kit/kit/ratelimit"
	httptransport "github.com/go-kit/kit/transport/http"
)

// ConcurlService provides operations on strings.
type ConcurlService interface {
	Get() (string, error)
}

type concurlService struct {
	logger  log.Logger
	payload string
	deps    []string
}

func (cs concurlService) Get() (string, error) {
	payload := cs.payload

	if len(cs.deps) > 0 {
		cs.logger.Log("proxy_to", fmt.Sprint(cs.deps))
		for _, backend := range cs.deps {
			var (
				qps         = 100 // max to each instance
				publisher   = static.NewPublisher([]string{backend}, factory(qps), cs.logger)
				lb          = loadbalancer.NewRoundRobin(publisher)
				maxAttempts = 3
				maxTime     = 100 * time.Millisecond
				endpoint    = loadbalancer.Retry(maxAttempts, maxTime, lb)
			)
			p := proxy{endpoint}
			body, err := p.Get()
			if err != nil {
				cs.logger.Log("err", err)
			}
			payload += body
		}
	}

	return payload, nil
}

type proxy struct {
	GetEndpoint endpoint.Endpoint
}

func (mw proxy) Get() (string, error) {
	ctx := context.Background()

	response, err := mw.GetEndpoint(ctx, getRequest{})
	if err != nil {
		return "", err
	}

	resp := response.(getResponse)
	if resp.Err != "" {
		return resp.Body, errors.New(resp.Err)
	}
	return resp.Body, nil
}

func factory(qps int) loadbalancer.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		var e endpoint.Endpoint
		e = makeGetProxy(instance)
		e = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(e)
		e = kitratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(float64(qps), int64(qps)))(e)
		return e, nil, nil
	}
}

func makeGetProxy(instance string) endpoint.Endpoint {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		panic(err)
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return httptransport.NewClient(
		"GET",
		u,
		encodeRequest,
		decodeGetResponse,
	).Endpoint()
}

// ServiceMiddleware is a chainable behavior modifier for ConcurlService.
type ServiceMiddleware func(ConcurlService) ConcurlService
