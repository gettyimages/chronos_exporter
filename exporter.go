package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jeffail/gabs"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

const namespace = "chronos"

type Exporter struct {
	scraper      Scraper
	mapper       *Mapper
	duration     prometheus.Gauge
	scrapeError  prometheus.Gauge
	totalErrors  prometheus.Counter
	totalScrapes prometheus.Counter
	Counters     *CounterContainer
	Gauges       *GaugeContainer
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	log.Debugln("Describing metrics")
	metricCh := make(chan prometheus.Metric)
	doneCh := make(chan struct{})

	go func() {
		for m := range metricCh {
			ch <- m.Desc()
		}
		close(doneCh)
	}()

	e.Collect(metricCh)
	close(metricCh)
	<-doneCh
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	log.Debugln("Collecting metrics")
	e.scrape(ch)

	ch <- e.duration
	ch <- e.totalScrapes
	ch <- e.totalErrors
	ch <- e.scrapeError
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) {
	e.totalScrapes.Inc()

	var err error
	defer func(begin time.Time) {
		e.duration.Set(time.Since(begin).Seconds())
		if err == nil {
			e.scrapeError.Set(0)
		} else {
			e.totalErrors.Inc()
			e.scrapeError.Set(1)
		}
	}(time.Now())

	content, err := e.scraper.Scrape()
	if err != nil {
		log.Debugf("Problem scraping metrics endpoint: %v\n", err)
		return
	}

	json, err := gabs.ParseJSON(content)
	if err != nil {
		log.Debugf("Problem parsing metrics response: %v\n", err)
		return
	}

	e.scrapeMetrics(json, ch)
}

func (e *Exporter) scrapeMetrics(json *gabs.Container, ch chan<- prometheus.Metric) {
	elements, _ := json.ChildrenMap()
	for key, element := range elements {
		switch key {
		case "message":
			log.Errorf("Problem collecting metrics: %s\n", element.Data().(string))
			return
		case "version":
			data := element.Data()
			version, ok := data.(string)
			if !ok {
				log.Errorf(fmt.Sprintf("Bad conversion! Unexpected value \"%v\" for version\n", data))
			} else {
				gauge, _ := e.Gauges.Fetch("metrics_version", "Chronos metrics version", "version")
				gauge.WithLabelValues(version).Set(1)
			}

		case "counters":
			e.scrapeCounters(element)
		case "gauges":
			e.scrapeGauges(element)
		case "histograms":
			e.scrapeHistograms(element)
		case "meters":
			e.scrapeMeters(element)
		case "timers":
			e.scrapeTimers(element)
		}
	}

	for _, counter := range e.Counters.counters {
		counter.Collect(ch)
	}
	for _, gauge := range e.Gauges.gauges {
		gauge.Collect(ch)
	}
}

func (e *Exporter) scrapeCounters(json *gabs.Container) {
	elements, _ := json.ChildrenMap()
	for key, element := range elements {
		new, err := e.scrapeCounter(key, element)
		if err != nil {
			log.Debug(err)
		} else if new {
			log.Infof("Added counter %q\n", key)
		}
	}
}

func (e *Exporter) scrapeCounter(metric string, json *gabs.Container) (bool, error) {
	data := json.Path("count").Data()
	count, ok := data.(float64)
	if !ok {
		return false, errors.New(fmt.Sprintf("Bad conversion! Unexpected value \"%v\" for counter %s\n", data, metric))
	}

	counter, new := e.mapper.counter(metric)
	counter.With(e.mapper.labels(metric)).Set(count)
	return new, nil
}

func (e *Exporter) scrapeGauges(json *gabs.Container) {
	elements, _ := json.ChildrenMap()
	for key, element := range elements {
		new, err := e.scrapeGauge(key, element)
		if err != nil {
			log.Debug(err)
		} else if new {
			log.Infof("Added gauge %q\n", key)
		}
	}
}

func (e *Exporter) scrapeGauge(metric string, json *gabs.Container) (bool, error) {
	data := json.Path("value").Data()
	value, ok := data.(float64)
	if !ok {
		return false, errors.New(fmt.Sprintf("Bad conversion! Unexpected value \"%v\" for gauge %s\n", data, metric))
	}

	gauge, new := e.mapper.counter(metric)
	gauge.With(e.mapper.labels(metric)).Set(value)
	return new, nil
}

func (e *Exporter) scrapeMeters(json *gabs.Container) {
	elements, _ := json.ChildrenMap()
	for key, element := range elements {
		new, err := e.scrapeMeter(key, element)
		if err != nil {
			log.Debug(err)
		} else if new {
			log.Infof("Added meter %q\n", key)
		}
	}
}

func (e *Exporter) scrapeMeter(metric string, json *gabs.Container) (bool, error) {
	count, ok := json.Path("count").Data().(float64)
	if !ok {
		return false, errors.New(fmt.Sprintf("Bad meter! %s has no count\n", metric))
	}
	units, ok := json.Path("units").Data().(string)
	if !ok {
		return false, errors.New(fmt.Sprintf("Bad meter! %s has no units\n", metric))
	}

	counter, rates, new := e.mapper.meter(metric, units)
	counter.WithLabelValues().Set(count)

	properties, _ := json.ChildrenMap()
	for key, property := range properties {
		if strings.Contains(key, "rate") {
			if value, ok := property.Data().(float64); ok {
				rates.WithLabelValues(renameRate(key)).Set(value)
			}
		}
	}

	return new, nil
}

func (e *Exporter) scrapeHistograms(json *gabs.Container) {
	elements, _ := json.ChildrenMap()
	for key, element := range elements {
		new, err := e.scrapeHistogram(key, element)
		if err != nil {
			log.Debug(err)
		} else if new {
			log.Infof("Added histogram %q\n", key)
		}
	}
}

func (e *Exporter) scrapeHistogram(metric string, json *gabs.Container) (bool, error) {
	count, ok := json.Path("count").Data().(float64)
	if !ok {
		return false, errors.New(fmt.Sprintf("Bad historgram! %s has no count\n", metric))
	}

	counter, percentiles, min, max, mean, stddev, new := e.mapper.histogram(metric)
	counter.With(e.mapper.labels(metric)).Set(count)

	properties, _ := json.ChildrenMap()
	for key, property := range properties {
		switch key {
		case "p50", "p75", "p95", "p98", "p99", "p999":
			if value, ok := property.Data().(float64); ok {
				percentiles.WithLabelValues(
					e.mapper.labelValues(metric, "0."+key[1:])...).Set(value)
			}
		case "min":
			if value, ok := property.Data().(float64); ok {
				min.With(e.mapper.labels(metric)).Set(value)
			}
		case "max":
			if value, ok := property.Data().(float64); ok {
				max.With(e.mapper.labels(metric)).Set(value)
			}
		case "mean":
			if value, ok := property.Data().(float64); ok {
				mean.With(e.mapper.labels(metric)).Set(value)
			}
		case "stddev":
			if value, ok := property.Data().(float64); ok {
				stddev.With(e.mapper.labels(metric)).Set(value)
			}
		}
	}

	return new, nil
}

func (e *Exporter) scrapeTimers(json *gabs.Container) {
	elements, _ := json.ChildrenMap()
	for key, element := range elements {
		new, err := e.scrapeTimer(key, element)
		if err != nil {
			log.Debug(err)
		} else if new {
			log.Infof("Added timer %q\n", key)
		}
	}
}

func (e *Exporter) scrapeTimer(metric string, json *gabs.Container) (bool, error) {
	count, ok := json.Path("count").Data().(float64)
	if !ok {
		return false, errors.New(fmt.Sprintf("Bad timer! %s has no count\n", metric))
	}
	units, ok := json.Path("rate_units").Data().(string)
	if !ok {
		return false, errors.New(fmt.Sprintf("Bad timer! %s has no units\n", metric))
	}

	counter, rates, percentiles, min, max, mean, stddev, new := e.mapper.timer(metric, units)
	counter.WithLabelValues().Set(count)

	properties, _ := json.ChildrenMap()
	for key, property := range properties {
		switch key {
		case "mean_rate", "m1_rate", "m5_rate", "m15_rate":
			if value, ok := property.Data().(float64); ok {
				rates.WithLabelValues(renameRate(key)).Set(value)
			}

		case "p50", "p75", "p95", "p98", "p99", "p999":
			if value, ok := property.Data().(float64); ok {
				percentiles.WithLabelValues("0." + key[1:]).Set(value)
			}
		case "min":
			if value, ok := property.Data().(float64); ok {
				min.WithLabelValues().Set(value)
			}
		case "max":
			if value, ok := property.Data().(float64); ok {
				max.WithLabelValues().Set(value)
			}
		case "mean":
			if value, ok := property.Data().(float64); ok {
				mean.WithLabelValues().Set(value)
			}
		case "stddev":
			if value, ok := property.Data().(float64); ok {
				stddev.WithLabelValues().Set(value)
			}
		}
	}

	return new, nil
}

func NewExporter(s Scraper) *Exporter {
	counters := NewCounterContainer()
	gauges := NewGaugeContainer()
	mapper := &Mapper{counters, gauges}
	return &Exporter{
		scraper:  s,
		mapper:   mapper,
		Counters: counters,
		Gauges:   gauges,
		duration: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "last_scrape_duration_seconds",
			Help:      "Duration of the last scrape of metrics from Chronos.",
		}),
		scrapeError: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "last_scrape_error",
			Help:      "Whether the last scrape of metrics from Chronos resulted in an error (1 for error, 0 for success).",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrapes_total",
			Help:      "Total number of times Chronos was scraped for metrics.",
		}),
		totalErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "errors_total",
			Help:      "Total number of times the exporter experienced errors collecting Chronos metrics.",
		}),
	}
}
