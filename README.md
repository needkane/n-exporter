# Prometheus Exporter
Exporter for Mesos master and agent metrics
Exporter for Consul server

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
```

Usually you would run one exporter with `-master` pointing to the current
leader and one exporter for each slave with `-slave` pointing to it. In
a default Mesos / DC/OS setup, you should be able to run the n-exporter
like this:

- Master: `n-exporter -master http://leader.mesos:5050`
- Agent: `n-exporter -slave http://localhost:5051`
