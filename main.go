package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

var (
	listenAddress = flag.String(
		"web.listen-address", ":9044",
		"Address to listen on for web interface and telemetry.")

	metricsPath = flag.String(
		"web.telemetry-path", "/metrics",
		"Path under which to expose metrics.")

	chronosUri = flag.String(
		"chronos.uri", "http://chronos.mesos:4400",
		"URI of Chronos")
)

func chronosConnect(uri *url.URL) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).Dial,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	response, err := client.Get(fmt.Sprintf("%v/ping", uri))
	if err != nil {
		log.Debugf("Problem connecting to Chronos: %v\n", err)
		return err
	}

	if response.StatusCode != 200 {
		log.Debugf("Problem reading Chronos ping response: %s\n", response.Status)
		return err
	}

	log.Debug("Connected to Chronos!")
	return nil
}

func main() {
	flag.Parse()
	uri, err := url.Parse(*chronosUri)
	if err != nil {
		log.Fatal(err)
	}

	retryTimeout := time.Duration(10 * time.Second)
	for {
		err := chronosConnect(uri)
		if err == nil {
			break
		}

		log.Debugf("Problem connecting to Chronos: %v", err)
		log.Infof("Couldn't connect to Chronos! Trying again in %v", retryTimeout)
		time.Sleep(retryTimeout)
	}

	exporter := NewExporter(&scraper{uri})
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
           <head><title>Chronos Exporter</title></head>
           <body>
           <h1>Chronos Exporter</h1>
           <p><a href='` + *metricsPath + `'>Metrics</a></p>
           </body>
           </html>`))
	})

	log.Info("Starting Server: ", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
