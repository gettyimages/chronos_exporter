# Chronos Prometheus Exporter

[![Build Status](https://travis-ci.org/gettyimages/chronos_exporter.svg?branch=master)](https://travis-ci.org/gettyimages/chronos_exporter)

A [Prometheus](http://prometheus.io) metrics exporter for the [Chronos](https://mesos.github.io/chronos) Mesos framework.

This exporter exposes Chronos' Codahale/Dropwizard metrics via its `/metrics` endpoint.

## Getting

```sh
$ go get github.com/gettyimages/chronos_exporter
```

*\-or-*

```sh
$ docker pull gettyimages/chronos_exporter
```

*\-or-*

```
make deps && make
bin/chronos_exporter --help
```

## Using

```sh
Usage of chronos_exporter:
  -chronos.uri string
        URI of Chronos (default "http://chronos.mesos:4400")
  -web.listen-address string
        Address to listen on for web interface and telemetry. (default ":9044")
  -web.telemetry-path string
        Path under which to expose metrics. (default "/metrics")
  -log.format value
        If set use a syslog logger or JSON logging. Example: logger:syslog?appname=bob&local=7 or logger:stdout?json=true. Defaults to stderr.
  -log.level value
        Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]. (default info)
```
