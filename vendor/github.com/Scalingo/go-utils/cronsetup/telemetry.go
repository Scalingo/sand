package cronsetup

import (
	"context"
	"time"

	otelsdk "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Scalingo/go-utils/cronsetup/internal/cron"
	"github.com/Scalingo/go-utils/errors/v3"
)

type telemetry struct {
	runsDuration metric.Float64Histogram
}

// jobNameAttributeKey captures the executed job name.
const jobNameAttributeKey = "scalingo.etcd_cron.job_name"

// statusAttributeKey captures the execution status.
const statusAttributeKey = "scalingo.etcd_cron.status"

const (
	statusSuccess = "success"
	statusError   = "error"
)

func newTelemetry(ctx context.Context) (*telemetry, error) {
	meter := otelsdk.Meter("scalingo.etcd_cron")

	runsDuration, err := meter.Float64Histogram(
		"scalingo.etcd_cron.run.duration",
		metric.WithDescription("Cron job execution duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "create runs duration histogram")
	}

	return &telemetry{
		runsDuration: runsDuration,
	}, nil
}

func (t *telemetry) wrapJob(job cron.Job) cron.Job {
	originalFunc := job.Func

	job.Func = func(ctx context.Context) error {
		startedAt := time.Now()
		err := originalFunc(ctx)

		status := statusSuccess
		if err != nil {
			status = statusError
		}
		attributes := metric.WithAttributes(
			attribute.String(jobNameAttributeKey, job.Name),
			attribute.String(statusAttributeKey, status),
		)
		t.runsDuration.Record(ctx, time.Since(startedAt).Seconds(), attributes)

		return err
	}

	return job
}
