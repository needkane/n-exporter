package prometheus

import (
	"bytes"
	"net/http"

	dto "github.com/prometheus/client_model/go"
)

type Registry interface {
	EnableCollectChecks(bool)
	PanicOnCollectError(bool)
	SetMetricFamilyInjectionHook(hook func() []*dto.MetricFamily)

	Register(c Collector) (Collector, error)
	MustRegister(c Collector) Collector
	RegisterOrGet(c Collector) (Collector, error)
	MustRegisterOrGet(c Collector) Collector
	Unregister(c Collector) bool

	Push(job, instance, url, method string) error
	PushOverride(job, instance, url string) error
	PushAdd(job, instance, url string) error
	PushDelete(job, instance, url string) error

	ServeHTTP(w http.ResponseWriter, req *http.Request)
}

var _ Registry = new(registry)

type NewRegistryOption struct {
	BufPool          chan *bytes.Buffer
	MetricFamilyPool chan *dto.MetricFamily
	MetricPool       chan *dto.Metric
	HttpClient       *http.Client
}

func NewRegistry(option NewRegistryOption) *registry {
	if option.HttpClient == nil {
		option.HttpClient = http.DefaultClient
	}
	return &registry{
		collectorsByID:   map[uint64]Collector{},
		descIDs:          map[uint64]struct{}{},
		dimHashesByName:  map[string]uint64{},
		bufPool:          option.BufPool,
		metricFamilyPool: option.MetricFamilyPool,
		metricPool:       option.MetricPool,
		httpClient:       option.HttpClient,
	}
}

func (r *registry) SetMetricFamilyInjectionHook(hook func() []*dto.MetricFamily) {
	r.metricFamilyInjectionHook = hook
}

func (r *registry) EnableCollectChecks(b bool) {
	r.collectChecksEnabled = b
}

func (r *registry) PanicOnCollectError(b bool) {
	r.panicOnCollectError = b
}

func (r *registry) MustRegister(c Collector) Collector {
	c, err := r.Register(c)
	if err != nil {
		panic(err)
	}
	return c
}

func (r *registry) MustRegisterOrGet(c Collector) Collector {
	c, err := r.RegisterOrGet(c)
	if err != nil {
		panic(err)
	}
	return c
}

func (r *registry) PushOverride(job, instance, url string) error {
	return r.Push(job, instance, url, "PUT")
}

func (r *registry) PushAdd(job, instance, url string) error {
	return r.Push(job, instance, url, "POST")
}

func (r *registry) PushDelete(job, instance, url string) error {
	return r.Push(job, instance, url, "DELETE")
}
