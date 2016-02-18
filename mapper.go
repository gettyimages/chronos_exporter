package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metric_symbol_repl_expr   = regexp.MustCompile(`[\.\$\-\(\)]`)
	chronos_jobs_capture_expr = regexp.MustCompile(`jobs\.run\.(\w+)\.([\w-]+)`)
)

type Mapper struct {
	Counters *CounterContainer
	Gauges   *GaugeContainer
}

func (m *Mapper) counter(metric string) (*prometheus.CounterVec, bool) {
	name, _ := renameMetric(metric)
	help := fmt.Sprintf(counterHelp, metric)
	return m.Counters.Fetch(name, help, m.labelKeys(metric)...)
}

func (m *Mapper) gauge(metric string) (*prometheus.GaugeVec, bool) {
	name, _ := renameMetric(metric)
	help := fmt.Sprintf(gaugeHelp, metric)
	return m.Gauges.Fetch(name, help)
}

func (m *Mapper) meter(metric string, units string) (*prometheus.CounterVec, *prometheus.GaugeVec, bool) {
	name, _ := renameMetric(metric)
	help := fmt.Sprintf(meterHelp, metric, units)
	counter, new := m.Counters.Fetch(name+"_count", help)

	rates, _ := m.Gauges.Fetch(name, help, "rate")
	return counter, rates, new
}

func (m *Mapper) histogram(metric string) (*prometheus.CounterVec, *prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.GaugeVec, bool) {
	name, _ := renameMetric(metric)
	help := fmt.Sprintf(histogramHelp, metric)
	counter, new := m.Counters.Fetch(name+"_count", help, m.labelKeys(metric)...)

	percentiles, _ := m.Gauges.Fetch(name, help, m.labelKeys(metric, "percentile")...)
	min, _ := m.Gauges.Fetch(name+"_min", help, m.labelKeys(metric)...)
	max, _ := m.Gauges.Fetch(name+"_max", help, m.labelKeys(metric)...)
	mean, _ := m.Gauges.Fetch(name+"_mean", help, m.labelKeys(metric)...)
	stddev, _ := m.Gauges.Fetch(name+"_stddev", help, m.labelKeys(metric)...)

	return counter, percentiles, min, max, mean, stddev, new
}

func (m *Mapper) timer(metric, units string) (*prometheus.CounterVec, *prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.GaugeVec, bool) {
	name, _ := renameMetric(metric)
	help := fmt.Sprintf(timerHelp, metric, units)
	counter, new := m.Counters.Fetch(name+"_count", help)

	rates, _ := m.Gauges.Fetch(name+"_rate", help, "rate")
	percentiles, _ := m.Gauges.Fetch(name, help, "percentile")
	min, _ := m.Gauges.Fetch(name+"_min", help)
	max, _ := m.Gauges.Fetch(name+"_max", help)
	mean, _ := m.Gauges.Fetch(name+"_mean", help)
	stddev, _ := m.Gauges.Fetch(name+"_stddev", help)

	return counter, rates, percentiles, min, max, mean, stddev, new
}

func (m *Mapper) labels(metric string) (labels map[string]string) {
	_, labels = renameMetric(metric)
	return
}

func (m *Mapper) labelKeys(metric string, extraKeys ...string) (keys []string) {
	labels := m.labels(metric)
	keys = make([]string, 0, len(labels)+len(extraKeys))
	for k := range labels {
		keys = append(keys, k)
	}
	for _, k := range extraKeys {
		keys = append(keys, k)
	}
	return
}

func (m *Mapper) labelValues(metric string, extraVals ...string) (vals []string) {
	labels := m.labels(metric)
	vals = make([]string, 0, len(labels)+len(extraVals))
	for _, v := range labels {
		vals = append(vals, v)
	}
	for _, v := range extraVals {
		vals = append(vals, v)
	}
	return
}

func renameMetric(name string) (string, map[string]string) {
	labels := map[string]string{}

	captures := chronos_jobs_capture_expr.FindStringSubmatch(name)
	if len(captures) == 3 {
		name = "jobs_run_" + captures[1]
		labels["job"] = captures[2]
	}

	name = metric_symbol_repl_expr.ReplaceAllLiteralString(name, "_")
	name = strings.TrimRight(name, "_")
	name = strings.ToLower(name)
	return name, labels
}

func renameRate(originalRate string) (name string) {
	switch originalRate {
	case "m1_rate":
		name = "1m"
	case "m5_rate":
		name = "5m"
	case "m15_rate":
		name = "15m"
	default:
		name = strings.TrimSuffix(originalRate, "_rate")
	}
	return
}
