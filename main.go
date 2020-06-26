package main

import (
	"flag"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/heptiolabs/healthcheck"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

var (
	hosturl       = flag.String(os.Getenv("AVI_CLUSTER"), "", "AVI Cluster URL.")
	listenAddress = flag.String("web.listen-address", ":9300", "Address to listen on for web interface and telemetry.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
)

const AVI_API_VERSION = "18.2.1"
const AVI_TENANT = "admin"

func main() {
	flag.Parse()

	username := os.Getenv("AVI_USERNAME")
	password := os.Getenv("AVI_PASSWORD")
	if username == "" {
		log.Fatalf("AVI_USERNAME environment variable must be set.")
	} else if password == "" {
		log.Fatalf("AVI_PASSWORD environment variable must be set.")
	}

	// Set various http endpoints.
	e := NewExporter()
	e.registerGauges()
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/avi", aviPromHTTPHandler(e, prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>AVI Exporter</title></head>
             <body>
             <h1>AVI Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	// Set service health endpoint.
	u, err := url.Parse(*hosturl)
	if err != nil {
		log.Fatalf("Could not parse host url: %s", err)
	}
	health := healthcheck.NewHandler()
	var port string
	if u.Port() == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	} else {
		port = u.Port()
	}

	health.AddReadinessCheck(
		"avi-tcp",
		healthcheck.Async(healthcheck.TCPDialCheck(u.Host+":"+port, 50*time.Millisecond), 10*time.Second))

	http.HandleFunc("/live", health.LiveEndpoint)
	http.HandleFunc("/healthz", health.ReadyEndpoint)
	//////////////////////////////////////////////////////////////////////////////
	log.Infoln("Starting HTTP server on", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatal(err)
	}
}
