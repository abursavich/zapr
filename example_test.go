package zapr_test

import (
	"flag"
	"net/http"

	"bursavich.dev/zapr"
	"bursavich.dev/zapr/zaprprom"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func ExampleNewLogger() {
	zaprObserver := zaprprom.NewObserver()
	zaprOptions := zapr.RegisterFlags(flag.CommandLine, zapr.AllOptions(
		zapr.WithObserver(zaprObserver),
		zapr.WithLevel(2), // Override default logging level.
	)...)
	addr := flag.String("http-address", ":8080", "HTTP server listen address.")
	flag.Parse()

	log, sink := zapr.NewLogger(zaprOptions...)
	defer sink.Flush() // For most GOOS (linux and darwin), flushing to stderr is a no-op.
	log.Info("Hello, zap logr with option flags!")

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewBuildInfoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		zaprObserver,
	)
	log.Info("Hello, zap logr Prometheus metrics!")

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	srv := http.Server{
		Addr:     *addr,
		Handler:  mux,
		ErrorLog: zapr.NewStdErrorLogger(sink), // Adapt LogSink to stdlib *log.Logger.
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Error(err, "Failed to serve HTTP")
	}
}
