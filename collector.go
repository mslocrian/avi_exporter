package main

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

type collector struct {
	ctx         context.Context
	controller  string
	tenant      string
	api_version string
	username    string
	password    string
	logger      log.Logger
}

func (c collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	metrics, err := CollectTarget(c.controller, c.username, c.password, c.tenant, c.api_version, c.logger)
	for _, metric := range metrics {
		// level.Info(c.logger).Log("metric", metric)
		ch <- metric
	}

	if err != nil {
		level.Info(c.logger).Log("msg", "Error scraping target", "err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("avi_error", "Error scraping target", nil, nil), err)
		return
	}
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("avi_scrape_duration_seocnds", "Total AVI time scrape took (query and processing).", nil, nil),
		prometheus.GaugeValue,
		time.Since(start).Seconds())
}

func (c collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}
