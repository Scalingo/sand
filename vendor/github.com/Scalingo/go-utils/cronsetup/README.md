# Package `cronsetup` v1.6.0

The `cronsetup` package eases the configuration and the execution of cron jobs on Go services. It has two modes: a distributed one (backed by etcd), and a local one.

## Goals

### Distributed Mode

The goal of the distributed mode is to implement a distributed and fault tolerant cron in order to:

* Run an identical process on several hosts
* Each of these process instantiate a cron with the same rules
* Ensure only one of these processes executes an iteration of a job

### Local Mode

The goal of the local mode is to implement a classical cron: it executes each job on all hosts executing the cron task.

## Examples

Examples are available in the `examples` package. One can execute them with:

```sh
go run ./examples/distributed
go run ./examples/local
```

The distributed example requires a etcd to be running at `127.0.0.1:2379`.

## Telemetry

Telemetry is enabled by default, and can be disabled with `WithoutTelemetry` set
to true in initialization option.

It records the duration of each job execution (in seconds). The metric uses the
`scalingo.etcd_cron.job_name` attribute with the job name as value, and a
`scalingo.etcd_cron.status` attribute with `success` or `error`.

Metrics:

- `scalingo.etcd_cron.run.duration`: execution time of a job in seconds
