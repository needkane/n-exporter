package main

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus.v2"
)

var elected string

func newMasterCollector(httpClient *httpClient) prometheus.Collector {
	metrics := map[prometheus.Collector]func(metricMap, prometheus.Collector) error{
		// Master stats about uptime and election state
		prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "mesos",
			Subsystem: "master",
			Name:      "elected",
			Help:      "1 if master is elected leader, 0 if not",
		}): func(m metricMap, c prometheus.Collector) error {
			electedValue, ok := m["master/elected"]
			if !ok {
				return notFoundInMap
			}
			c.(prometheus.Gauge).Set(electedValue)
			elected = strconv.FormatFloat(electedValue, 'f', 10, 32)
			return nil
		},
		// CPU/Disk/Mem resources in free/used
		gauge("master", "cpus", "Current CPU resources in cluster.", "elected", "type"): func(m metricMap, c prometheus.Collector) error {
			total, ok := m["master/cpus_total"]
			used, ok := m["master/cpus_used"]
			if !ok {
				return notFoundInMap
			}
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "cpu_free").Set(total - used)
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "cpu_used").Set(used)
			return nil
		},
		gauge("master", "mem", "Current memory resources in cluster.", "elected", "type"): func(m metricMap, c prometheus.Collector) error {
			total, ok := m["master/mem_total"]
			used, ok := m["master/mem_used"]
			if !ok {
				return notFoundInMap
			}
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "mem_free").Set(total - used)
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "mem_used").Set(used)
			return nil
		},
		gauge("master", "disk", "Current disk resources in cluster.", "elected", "type"): func(m metricMap, c prometheus.Collector) error {
			total, ok := m["master/disk_total"]
			used, ok := m["master/disk_used"]
			if !ok {
				return notFoundInMap
			}
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "disk_free").Set(total - used)
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "disk_used").Set(used)
			return nil
		},
		prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "mesos",
			Subsystem: "master",
			Name:      "uptime_seconds",
			Help:      "Number of seconds the master process is running.",
		}): func(m metricMap, c prometheus.Collector) error {
			uptime, ok := m["master/uptime_secs"]
			if !ok {
				return notFoundInMap
			}
			c.(prometheus.Gauge).Set(uptime)
			return nil
		},
		// Master stats about agents
		counter("master", "slave_registration_events_total", "Total number of registration events on this master since it booted.", "elected", "event"): func(m metricMap, c prometheus.Collector) error {
			registrations, ok := m["master/slave_registrations"]
			reregistrations, ok := m["master/slave_reregistrations"]
			if !ok {
				return notFoundInMap
			}
			c.(*settableCounterVec).Set(registrations, elected, "register")
			c.(*settableCounterVec).Set(reregistrations, elected, "reregister")
			return nil
		},

		counter("master", "slave_removal_events_total", "Total number of removal events on this master since it booted.", "elected", "event"): func(m metricMap, c prometheus.Collector) error {
			scheduled, ok := m["master/slave_shutdowns_scheduled"]
			canceled, ok := m["master/slave_shutdowns_canceled"]
			completed, ok := m["master/slave_shutdowns_completed"]
			removals, ok := m["master/slave_removals"]
			if !ok {
				return notFoundInMap
			}

			c.(*settableCounterVec).Set(scheduled, elected, "scheduled")
			c.(*settableCounterVec).Set(canceled, elected, "canceled")
			c.(*settableCounterVec).Set(completed, elected, "completed")
			c.(*settableCounterVec).Set(removals-completed, elected, "died")
			return nil
		},
		gauge("master", "slaves_state", "Current number of slaves known to the master per connection and registration state.", "elected", "connection_state", "registration_state"): func(m metricMap, c prometheus.Collector) error {
			active, ok := m["master/slaves_active"]
			inactive, ok := m["master/slaves_inactive"]
			disconnected, ok := m["master/slaves_disconnected"]

			if !ok {
				return notFoundInMap
			}
			// FIXME: Make sure those assumptions are right
			// Every "active" node is connected to the master
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "connected", "active").Set(active)
			// Every "inactive" node is connected but node sending offers
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "connected", "inactive").Set(inactive)
			// Every "disconnected" node is "inactive"
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "disconnected", "inactive").Set(disconnected)
			// Every "connected" node is either active or inactive
			return nil
		},

		// Master stats about frameworks
		gauge("master", "frameworks_state", "Current number of frames known to the master per connection and registration state.", "elected", "connection_state", "registration_state"): func(m metricMap, c prometheus.Collector) error {
			active, ok := m["master/frameworks_active"]
			inactive, ok := m["master/frameworks_inactive"]
			disconnected, ok := m["master/frameworks_disconnected"]

			if !ok {
				return notFoundInMap
			}
			// FIXME: Make sure those assumptions are right
			// Every "active" framework is connected to the master
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "connected", "active").Set(active)
			// Every "inactive" framework is connected but framework sending offers
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "connected", "inactive").Set(inactive)
			// Every "disconnected" framework is "inactive"
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "disconnected", "inactive").Set(disconnected)
			// Every "connected" framework is either active or inactive
			return nil
		},
		prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "mesos",
			Subsystem: "master",
			Name:      "offers_pending",
			Help:      "Current number of offers made by the master which aren't yet accepted or declined by frameworks.",
		}): func(m metricMap, c prometheus.Collector) error {
			offers, ok := m["master/outstanding_offers"]
			if !ok {
				return notFoundInMap
			}
			c.(prometheus.Gauge).Set(offers)
			// c.(*prometheus.Gauge).Set(offers)
			return nil
		},
		// Master stats about tasks
		counter("master", "task_states_exit_total", "Total number of tasks processed by exit state.", "elected", "state"): func(m metricMap, c prometheus.Collector) error {
			errored, ok := m["master/tasks_error"]
			failed, ok := m["master/tasks_failed"]
			finished, ok := m["master/tasks_finished"]
			killed, ok := m["master/tasks_killed"]
			lost, ok := m["master/tasks_lost"]
			if !ok {
				return notFoundInMap
			}
			c.(*settableCounterVec).Set(errored, elected, "errored")
			c.(*settableCounterVec).Set(failed, elected, "failed")
			c.(*settableCounterVec).Set(finished, elected, "finished")
			c.(*settableCounterVec).Set(killed, elected, "killed")
			c.(*settableCounterVec).Set(lost, elected, "lost")
			return nil
		},
		counter("master", "task_states_current", "Current number of tasks by state.", "elected", "state"): func(m metricMap, c prometheus.Collector) error {
			running, ok := m["master/tasks_running"]
			staging, ok := m["master/tasks_staging"]
			starting, ok := m["master/tasks_starting"]
			if !ok {
				return notFoundInMap
			}
			c.(*settableCounterVec).Set(running, elected, "running")
			c.(*settableCounterVec).Set(staging, elected, "staging")
			c.(*settableCounterVec).Set(starting, elected, "starting")
			return nil
		},

		// Master stats about messages
		counter("master", "messages_outcomes_total",
			"Total number of messages by outcome of operation and direction.", "elected",
			"source", "destination", "type", "outcome"): func(m metricMap, c prometheus.Collector) error {
			frameworkToExecutorValid, ok := m["master/valid_framework_to_executor_messages"]
			frameworkToExecutorInvalid, ok := m["master/invalid_framework_to_executor_messages"]
			executorToFrameworkValid, ok := m["master/valid_executor_to_framework_messages"]
			executorToFrameworkInvalid, ok := m["master/invalid_executor_to_framework_messages"]

			// status updates are sent from framework?(FIXME) to slave
			// status update acks are sent from slave to framework?
			statusUpdateAckValid, ok := m["master/valid_status_update_acknowledgements"]
			statusUpdateAckInvalid, ok := m["master/invalid_status_update_acknowledgements"]
			statusUpdateValid, ok := m["master/valid_status_updates"]
			statusUpdateInvalid, ok := m["master/invalid_status_updates"]

			if !ok {
				return notFoundInMap
			}
			c.(*settableCounterVec).Set(frameworkToExecutorValid, elected, "framework", "executor", "", "valid")
			c.(*settableCounterVec).Set(frameworkToExecutorInvalid, elected, "framework", "executor", "", "invalid")

			c.(*settableCounterVec).Set(executorToFrameworkValid, elected, "executor", "framework", "", "valid")
			c.(*settableCounterVec).Set(executorToFrameworkInvalid, elected, "executor", "framework", "", "invalid")

			// We consider a ack message simply as a message from slave to framework
			c.(*settableCounterVec).Set(statusUpdateValid, elected, "framework", "slave", "status_update", "valid")
			c.(*settableCounterVec).Set(statusUpdateInvalid, elected, "framework", "slave", "status_update", "invalid")
			c.(*settableCounterVec).Set(statusUpdateAckValid, elected, "slave", "framework", "status_update", "valid")
			c.(*settableCounterVec).Set(statusUpdateAckInvalid, elected, "slave", "framework", "status_update", "invalid")
			return nil
		},
		counter("master", "messages_type_total", "Total number of valid messages by type.", "elected", "type"): func(m metricMap, c prometheus.Collector) error {
			for k, v := range m {
				i := strings.Index("master/messages_", k)
				if i == -1 {
					continue
				}
				// FIXME: We expose things like messages_framework_to_executor twice
				c.(*settableCounterVec).Set(v, elected, k[i:])
			}
			return nil
		},

		// Master stats about events
		gauge("master", "event_queue_length", "Current number of elements in event queue by type", "elected", "type"): func(m metricMap, c prometheus.Collector) error {
			dispatches, ok := m["master/event_queue_dispatches"]
			httpRequests, ok := m["master/event_queue_http_requests"]
			messages, ok := m["master/event_queue_messages"]
			if !ok {
				return notFoundInMap
			}

			c.(*prometheus.GaugeVec).WithLabelValues(elected, "message").Set(messages)
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "http_request").Set(httpRequests)
			c.(*prometheus.GaugeVec).WithLabelValues(elected, "dispatches").Set(dispatches)
			return nil
		},

		// Master stats about registrar
		prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "mesos",
			Subsystem: "master",
			Name:      "state_store_seconds",
			Help:      "Registry write latency in seconds",
		}): func(m metricMap, c prometheus.Collector) error {
			//	c.(*prometheus.Histogram).Buckets //FIXME
			return nil
		},
	}
	return newMetricCollector(httpClient, metrics)
}
