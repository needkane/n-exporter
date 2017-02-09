package common

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus.v2"
)

type (
	label struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	status struct {
		State     string  `json:"state"`
		Timestamp float64 `json:"timestamp"`
	}
)

type metricMap map[string]float64

var (
	notFoundInMap = errors.New("Couldn't find key in map")
)

type SettableCounterVec struct {
	desc   *prometheus.Desc
	values []prometheus.Metric
}

func (c *SettableCounterVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *SettableCounterVec) Collect(ch chan<- prometheus.Metric) {
	for _, v := range c.values {
		ch <- v
	}

	c.values = nil
}

func (c *SettableCounterVec) Set(value float64, labelValues ...string) {
	c.values = append(c.values, prometheus.MustNewConstMetric(c.desc, prometheus.CounterValue, value, labelValues...))
}

func (c *SettableCounterVec) SetGauge(value float64, labelValues ...string) {
	c.values = append(c.values, prometheus.MustNewConstMetric(c.desc, prometheus.CounterValue, value, labelValues...))
}

type settableCounter struct {
	desc  *prometheus.Desc
	value prometheus.Metric
}

func (c *settableCounter) Describe(ch chan<- *prometheus.Desc) {
	if c.desc == nil {
		log.Printf("NIL description: %v", c)
	}
	ch <- c.desc
}

func (c *settableCounter) Collect(ch chan<- prometheus.Metric) {
	if c.value == nil {
		log.Printf("NIL value: %v", c)
	}
	ch <- c.value
}

func (c *settableCounter) Set(value float64) {
	c.value = prometheus.MustNewConstMetric(c.desc, prometheus.CounterValue, value)
}

func newSettableCounter(subsystem, name, help string) *settableCounter {
	return &settableCounter{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName("mesos", subsystem, name),
			help,
			nil,
			prometheus.Labels{},
		),
	}
}

func gauge(subsystem, name, help string, labels ...string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "mesos",
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	}, labels)
}

func Counter(subsystem, name, help string, labels ...string) *SettableCounterVec {
	desc := prometheus.NewDesc(
		prometheus.BuildFQName("mesos", subsystem, name),
		help,
		labels,
		prometheus.Labels{},
	)

	return &SettableCounterVec{
		desc:   desc,
		values: nil,
	}
}

func CustomCounter(namespace, name, help string, labels ...string) *SettableCounterVec {
	desc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", name),
		help,
		labels,
		prometheus.Labels{},
	)

	return &SettableCounterVec{
		desc:   desc,
		values: nil,
	}
}

type authInfo struct {
	username string
	password string
}

type httpClient struct {
	http.Client
	url  string
	auth authInfo
}

type metricCollector struct {
	*httpClient
	metrics map[prometheus.Collector]func(metricMap, prometheus.Collector) error
}

func newMetricCollector(httpClient *httpClient, metrics map[prometheus.Collector]func(metricMap, prometheus.Collector) error) prometheus.Collector {
	return &metricCollector{httpClient, metrics}
}

func (httpClient *httpClient) fetchAndDecode(endpoint string, target interface{}) bool {
	url := strings.TrimSuffix(httpClient.url, "/") + endpoint
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating HTTP request to %s: %s", url, err)
		return false
	}
	if httpClient.auth.username != "" && httpClient.auth.password != "" {
		req.SetBasicAuth(httpClient.auth.username, httpClient.auth.password)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error fetching %s: %s", url, err)
		return false
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&target); err != nil {
		log.Printf("Error decoding response body from %s: %s", url, err)
		return false
	}

	return true
}

func (c *metricCollector) Collect(ch chan<- prometheus.Metric) {
	var m metricMap
	c.fetchAndDecode("/metrics/snapshot", &m)
	for cm, f := range c.metrics {
		if err := f(m, cm); err != nil {
			if err == notFoundInMap {
				ch := make(chan *prometheus.Desc, 1)
				cm.Describe(ch)
				log.Printf("Couldn't find fields required to update %s\n", <-ch)
			} else {
				log.Printf("Error extracting metric: %s", err)
			}
			continue
		}
		cm.Collect(ch)
	}
}

func (c *metricCollector) Describe(ch chan<- *prometheus.Desc) {
	for m, _ := range c.metrics {
		m.Describe(ch)
	}
}
