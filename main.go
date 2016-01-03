package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	httptransport "github.com/go-kit/kit/transport/http"
)

var (
	globalFlagset = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	globalFlags   = struct {
		debug   bool
		version bool
		dep     string
		payload string
		port    string
	}{}

	projectVersion = "dev"
	projectBuild   string
)

func init() {
	globalFlagset.BoolVar(&globalFlags.debug, "debug", false, "Print out more debug information to stderr")
	globalFlagset.BoolVar(&globalFlags.version, "version", false, "Print the version and exit")
	globalFlagset.StringVar(&globalFlags.dep, "dep", "", "Optional comma-separated list of other concurl instances to request data from")
	globalFlagset.StringVar(&globalFlags.payload, "payload", "", "Optional payload to send back as a response")
	globalFlagset.StringVar(&globalFlags.port, "port", ":80", "Optional port listen on")
}

func main() {
	globalFlagset.Parse(os.Args[1:])

	// deal specially with --version
	if globalFlags.version {
		fmt.Println(os.Args[0], "version", projectVersion, projectBuild)
		os.Exit(0)
	}

	var logger log.Logger
	logger = log.NewLogfmtLogger(os.Stderr)
	logger = log.NewContext(logger).With("dep", globalFlags.dep).With("payload", globalFlags.payload).With("port", globalFlags.port).With("caller", log.DefaultCaller)

	ctx := context.Background()

	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounter(stdprometheus.CounterOpts{
		Namespace: "my_group",
		Subsystem: "concurl_service",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestLatency := metrics.NewTimeHistogram(time.Microsecond, kitprometheus.NewSummary(stdprometheus.SummaryOpts{
		Namespace: "my_group",
		Subsystem: "concurl_service",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys))
	countResult := kitprometheus.NewSummary(stdprometheus.SummaryOpts{
		Namespace: "my_group",
		Subsystem: "concurl_service",
		Name:      "count_result",
		Help:      "The result of each count method.",
	}, []string{})

	var svc ConcurlService
	svc = concurlService{
		logger:  logger,
		payload: globalFlags.payload,
		deps:    strings.Split(globalFlags.dep, ","),
	}
	svc = loggingMiddleware(logger)(svc)
	svc = instrumentingMiddleware(requestCount, requestLatency, countResult)(svc)

	getHandler := httptransport.NewServer(
		ctx,
		makeGetEndpoint(svc),
		decodeGetRequest,
		encodeResponse,
	)

	http.Handle("/", getHandler)
	http.Handle("/metrics", stdprometheus.Handler())

	s := &http.Server{
		Addr:           globalFlags.port,
		Handler:        nil,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	logger.Log("msg", "HTTP", "addr", globalFlags.port)
	l, e := net.Listen("tcp", globalFlags.port)
	if e != nil {
		logger.Log("err", e.Error())
	}
	go s.Serve(l)

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Log("signal", <-ch)

	l.Close()
}
