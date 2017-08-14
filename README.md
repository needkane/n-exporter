# Prometheus Exporter
Exporter for Mesos master and agent metrics
Exporter for Consul server

[![version](https://img.shields.io/github/tag/needkane/n-exporter.svg)](https://github.com/needkane/n-exporter/releases/latest)
[![API Reference](
https://godoc.org/github.com/needkane/n-exporter?status.svg)](https://godoc.org/github.com/tendermint/tendermint)
[![license](https://img.shields.io/github/license/needkane/n-exporter.svg)](https://github.com/needkane/n-exporter/blob/master/LICENSE)
[![](https://tokei.rs/b1/github/needkane/n-exporter?category=lines)](https://github.com/needkane/n-exporter)


Branch    | Buid Status | Coverage | Report Card
----------|-------|----------|-------------
develop   |
[![Build Status](https://travis-ci.org/needknae/n-exporter.svg?branch=develop)](https://travis-ci.org/needkane/n-exporter) | [![codecov](https://codecov.io/gh/needkane/n-exporter/branch/develop/graph/badge.svg)](https://codecov.io/gh/needkane/n-exporter) | [![Go Report Card](https://goreportcard.com/badge/github.com/needkane/n-exporter/tree/develop)](https://goreportcard.com/report/github.com/needkane/n-exporter/tree/develop)
master    | 
[![Build Status](https://travis-ci.org/needkane/n-exporter.svg?branch=master)](https://travis-ci.org/needkane/n-exporter)| [![codecov](https://codecov.io/gh/needkane/n-exporter/branch/master/graph/badge.svg)](https://codecov.io/gh/needkane/n-exporter) | [![Go Report Card](https://goreportcard.com/badge/github.com/needkane/n-exporter/tree/master)](https://goreportcard.com/report/github.com/needkane/n-exporter/tree/master)
## Installing
```sh
$ go get github.com/needkane/n-exporter
```

## Using
The Mesos Exporter can either expose cluster wide metrics from a master or task
metrics from an agent.
The Consul Exporter can pull data from Consul Server

```sh
Usage of n-exporter:
  -addr string
       	Address to listen on (default ":9110")
  -ignoreCompletedFrameworkTasks
       	Don't export task_state_time metric
  -master string
       	Expose metrics from master running on this URL
  -slave string
       	Expose metrics from slave running on this URL
  -timeout duration
       	Master polling timeout (default 5s)
  -target_interval 
        interval for each target,if targetNo == 2 ,then the same target interval is 2 * target_interval
```

Usually you would run one exporter with `-master` pointing to the current
leader and one exporter for each slave with `-slave` pointing to it. In
a default Mesos / DC/OS setup, you should be able to run the n-exporter
like this:

- Master: `n-exporter -master http://leader.mesos:5050`
- Agent: `n-exporter -slave http://localhost:5051`
