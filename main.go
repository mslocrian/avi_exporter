package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/version"
)

const (
	AVI_API_VERSION = "18.2.1"
	AVI_TENANT      = "*"
)

var (
	hosturl       = flag.String(os.Getenv("AVI_CLUSTER"), "", "AVI Cluster URL.")
	listenAddress = flag.String("web.listen-address", ":9300", "Address to listen on for web interface and telemetry.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")

	aviDuration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "avi_collection_durations_seconds",
			Help: "Duration of collections by the AVI exporter",
		},
		[]string{"controller"},
	)
	aviRequestErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "avi_request_errors_total",
			Help: "Errors in requests to the AVI exporter",
		},
	)
)

func init() {
	prometheus.MustRegister(aviDuration)
	prometheus.MustRegister(aviRequestErrors)
	prometheus.MustRegister(version.NewCollector("avi_exporter"))
}

func handler(w http.ResponseWriter, r *http.Request, username string, password string, logger log.Logger) {
	query := r.URL.Query()

	// Gather controller name from query string
	controller := query.Get("controller")
	if (len(query["controller"]) != 1) || (controller == "") {
		http.Error(w, "'controller' parameter must be specified once.", 400)
		aviRequestErrors.Inc()
		return
	}

	// Gather tenant information from query string
	tenant := query.Get("tenant")
	if len(query["tenant"]) > 1 {
		http.Error(w, "'tenant' parameter can only be specified once.", 400)
		aviRequestErrors.Inc()
		return
	}

	if tenant == "" {
		tenant = AVI_TENANT
	}

	// Gather AVI API Version from query string
	api_version := query.Get("api_version")
	if len(query["api_version"]) > 1 {
		http.Error(w, "'api_version' parameter can only be specified once.", 400)
		aviRequestErrors.Inc()
		return
	}

	if api_version == "" {
		api_version = AVI_API_VERSION
	}

	logger = log.With(logger, "controller", controller, "tenant", tenant, "api_version", api_version)
	level.Debug(logger).Log("msg", "Starting scrape")

	start := time.Now()
	registry := prometheus.NewRegistry()
	collector := collector{ctx: r.Context(),
		controller:  controller,
		tenant:      tenant,
		api_version: api_version,
		username:    username,
		password:    password,
		logger:      logger}
	registry.MustRegister(collector)
	// Delegate http serving to Prometheus client library, which will calll collector.Collect
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	duration := time.Since(start).Seconds()
	aviDuration.WithLabelValues(controller).Observe(duration)
	level.Debug(logger).Log("msg", "Finished scrape", "duration_seconds", duration)
}

func main() {
	logLevel := &promlog.AllowedLevel{}
	logLevel.Set("debug")
	logFormat := &promlog.AllowedFormat{}

	promlogConfig := &promlog.Config{Level: logLevel, Format: logFormat}
	flag.Parse()
	logger := promlog.New(promlogConfig)

	username := os.Getenv("AVI_USERNAME")
	password := os.Getenv("AVI_PASSWORD")

	if username == "" {
		level.Error(logger).Log("msg", "AVI_USERNAME environment variable must be set.")
		os.Exit(1)
	} else if password == "" {
		level.Error(logger).Log("msg", "AVI_PASSWORD environment variable must be set.")
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "Starting avi_exporter", "version", version.Info())
	level.Info(logger).Log("build_context", version.BuildContext())

	// Set various http endpoints.
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/avi", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, username, password, logger)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>AVI Exporter</title></head>
             <body>
             <h1>AVI Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	level.Info(logger).Log("msg", "Listening on address", "address", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
		os.Exit(1)
	}
}
