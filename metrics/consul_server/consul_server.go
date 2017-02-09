package consul_server

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	consul_api "github.com/hashicorp/consul/api"
	"github.com/needkane/n-exporter/common"
	"github.com/prometheus/client_golang/prometheus.v2"
)

type AuthInfo struct {
	Username string
	Password string
}

type HttpClient struct {
	http.Client
	Url  string
	Auth AuthInfo
}
type metricMap map[string]float64
type (
	consulServerCollector struct {
		*HttpClient
		client  *consul_api.Client
		metrics map[prometheus.Collector]func(metricMap, prometheus.Collector) error
	}
)

var consulNameSpace = "consul"
var catalog_services_num = "catalog_services_num"

func NewConsulServerCollector(uri string, httpClient *HttpClient) (prometheus.Collector, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid consul URL: %s", err)
	}
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("invalid consul URL: %s", uri)
	}

	config := consul_api.DefaultConfig()
	config.Address = u.Host
	config.Scheme = u.Scheme

	client, err := consul_api.NewClient(config)
	if err != nil {
		return nil, err

	}

	metrics := map[prometheus.Collector]func(metricMap, prometheus.Collector) error{
		common.CustomCounter(consulNameSpace, catalog_services_num, "How many services are in the cluster.", "catalog_services_num"): func(m metricMap, c prometheus.Collector) error {
			servicesNum, ok := m[catalog_services_num]
			if !ok {
				return fmt.Errorf("notFoundInMap")
			}
			log.Println("consul_server.go 63------", servicesNum)
			c.(*common.SettableCounterVec).SetGauge(servicesNum, "servicesNum")
			return nil
		},
	}

	return &consulServerCollector{
		HttpClient: httpClient,
		metrics:    metrics,
		client:     client,
	}, nil
}

func (c *consulServerCollector) Collect(ch chan<- prometheus.Metric) {
	//c.fetchAndDecode("/v1/catalog/services", &s)
	// Query for the full list of services.
	serviceNames, _, err := c.client.Catalog().Services(&consul_api.QueryOptions{})
	if err != nil {
		fmt.Errorf("Catalog().Services() failed")
	}
	m := metricMap{catalog_services_num: float64(len(serviceNames))}
	for c, set := range c.metrics {
		set(m, c)
		c.Collect(ch)
	}
}

func (c *consulServerCollector) Describe(ch chan<- *prometheus.Desc) {
	for metric := range c.metrics {
		metric.Describe(ch)
	}
}

type ranges [][2]uint64

func (rs *ranges) UnmarshalJSON(data []byte) (err error) {
	if data = bytes.Trim(data, `[]"`); len(data) == 0 {
		return nil
	}

	var rng [2]uint64
	for _, r := range bytes.Split(data, []byte(",")) {
		ps := bytes.SplitN(r, []byte("-"), 2)
		if len(ps) != 2 {
			return fmt.Errorf("bad range: %s", r)
		}

		rng[0], err = strconv.ParseUint(string(bytes.TrimSpace(ps[0])), 10, 64)
		if err != nil {
			return err
		}

		rng[1], err = strconv.ParseUint(string(bytes.TrimSpace(ps[1])), 10, 64)
		if err != nil {
			return err
		}

		*rs = append(*rs, rng)
	}

	return nil
}

func (rs ranges) size() uint64 {
	var sz uint64
	for i := range rs {
		sz += 1 + (rs[i][1] - rs[i][0])
	}
	return sz
}
