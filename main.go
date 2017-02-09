package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/needkane/n-exporter/metrics/consul_server"
	"github.com/prometheus/client_golang/prometheus.v2"
)

var errorCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: "mesos",
	Subsystem: "collector",
	Name:      "errors_total",
	Help:      "Total number of internal mesos-collector errors.",
})

func getX509CertPool(pemFiles []string) *x509.CertPool {
	pool := x509.NewCertPool()
	for _, f := range pemFiles {
		content, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		ok := pool.AppendCertsFromPEM(content)
		if !ok {
			log.Fatal("Error parsing .pem file %s", f)
		}
	}
	return pool
}
func mkConsulHttpClient(url string, timeout time.Duration, auth consul_server.AuthInfo, certPool *x509.CertPool) *consul_server.HttpClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: certPool},
	}
	return &consul_server.HttpClient{
		http.Client{Timeout: timeout, Transport: transport},
		url,
		auth,
	}
}

func mkHttpClient(url string, timeout time.Duration, auth authInfo, certPool *x509.CertPool) *httpClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: certPool},
	}
	return &httpClient{
		http.Client{Timeout: timeout, Transport: transport},
		url,
		auth,
	}
}

func main() {
	fs := flag.NewFlagSet("n-exporter", flag.ExitOnError)
	addr := fs.String("addr", ":9111", "Address to listen on")
	masterURL := fs.String("master", "", "Expose metrics from master running on this URL")
	consulServer := fs.String("consulServer", "", "Expose metrics from consulServer")
	slaveURL := fs.String("agent", "", "Expose metrics from slave running on this URL")
	timeout := fs.Duration("timeout", 5*time.Second, "Master polling timeout")
	exportedTaskLabels := fs.String("exportedTaskLabels", "", "Comma-separated list of task labels to include in the task_labels metric")
	ignoreCompletedFrameworkTasks := fs.Bool("ignoreCompletedFrameworkTasks", false, "Don't export task_state_time metric")
	trustedCerts := fs.String("trustedCerts", "", "Comma-separated list of certificates (.pem files) trusted for requests to Mesos endpoints")

	fs.Parse(os.Args[1:])

	auth := authInfo{
		os.Getenv("MESOS_EXPORTER_USERNAME"),
		os.Getenv("MESOS_EXPORTER_PASSWORD"),
	}

	Auth := consul_server.AuthInfo{
		os.Getenv("MESOS_EXPORTER_USERNAME"),
		os.Getenv("MESOS_EXPORTER_PASSWORD"),
	}

	var certPool *x509.CertPool = nil
	if *trustedCerts != "" {
		certPool = getX509CertPool(strings.Split(*trustedCerts, ","))
	}

	if *consulServer != "" {
		reg := prometheus.NewCustomRegistry()
		if _, err := reg.Register(errorCounter); err != nil {
			log.Fatal(err)
		}
		for _, f := range []func(*consul_server.HttpClient) prometheus.Collector{
			func(c *consul_server.HttpClient) prometheus.Collector {
				coll, err := consul_server.NewConsulServerCollector(*consulServer, c)
				if err != nil {
					log.Fatal(err)
				}
				return coll
			},
		} {
			c := f(mkConsulHttpClient(*consulServer, *timeout, Auth, certPool))
			if _, err := reg.Register(c); err != nil {
				log.Fatal(err)
			}
		}
		log.Printf("Exposing master metrics on %s", *addr)
		http.Handle("/metrics/consul-server", reg.Handler())
	}

	if *masterURL != "" {
		reg := prometheus.NewCustomRegistry()
		if _, err := reg.Register(errorCounter); err != nil {
			log.Fatal(err)
		}
		for _, f := range []func(*httpClient) prometheus.Collector{
			newMasterCollector,
			func(c *httpClient) prometheus.Collector {
				return newMasterStateCollector(c, *ignoreCompletedFrameworkTasks)
			},
		} {
			c := f(mkHttpClient(*masterURL, *timeout, auth, certPool))
			if _, err := reg.Register(c); err != nil {
				log.Fatal(err)
			}
		}
		log.Printf("Exposing master metrics on %s", *addr)
		http.Handle("/metrics/mesos-master", reg.Handler())
	}

	if *slaveURL != "" {
		reg := prometheus.NewCustomRegistry()
		if _, err := reg.Register(errorCounter); err != nil {
			log.Fatal(err)
		}
		slaveCollectors := []func(*httpClient) prometheus.Collector{
			func(c *httpClient) prometheus.Collector {
				return newSlaveCollector(c)
			},
			func(c *httpClient) prometheus.Collector {
				return newSlaveMonitorCollector(c)
			},
		}
		if *exportedTaskLabels != "" {
			slaveLabels := strings.Split(*exportedTaskLabels, ",")
			slaveCollectors = append(slaveCollectors, func(c *httpClient) prometheus.Collector {
				return newSlaveStateCollector(c, slaveLabels)
			})
		}

		for _, f := range slaveCollectors {
			c := f(mkHttpClient(*slaveURL, *timeout, auth, certPool))
			if _, err := reg.Register(c); err != nil {
				log.Fatal(err)
			}
		}
		log.Printf("Exposing slave metrics on %s", *addr)
		http.Handle("/metrics/mesos-agent", reg.Handler())
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head><title>Needkane Exporter</title></head>
            <body>
            <h1>Needkane Exporter</h1>
            <p><a href="/metrics/mesos-agent">MetricsMesosAgent</a></p>
            <p><a href="/metrics/mesos-master">MetricsMesosMaster</a></p>
            <p><a href="/metrics/consul-server">MetricsConsulServer</a></p>
            </body>
            </html>`))
	})
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}
