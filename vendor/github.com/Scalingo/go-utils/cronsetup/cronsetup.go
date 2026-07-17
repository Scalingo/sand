package cronsetup

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/sirupsen/logrus"
	etcdv3 "go.etcd.io/etcd/client/v3"

	"github.com/Scalingo/go-utils/cronsetup/internal/cron"
	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
)

// Job represents a cron job. It contains 3 *mandatory* options to define a job.
type Job = cron.Job

// SetupOpts are the options to setup new cron jobs. One of EtcdClient or EtcdConfig must be provided.
type SetupOpts struct {
	// EtcdClient is the etcd client to use to set mutexes
	EtcdClient *etcdv3.Client
	// EtcdConfig is the configuration to use in order to create and etcd client to set mutexes
	EtcdConfig func() (etcdv3.Config, error)
	// List of jobs to execute
	Jobs []Job
	// WithoutTelemetry indicates whether OpenTelemetry instrumentation should be disabled
	WithoutTelemetry bool
}

// Setup configures a new etcd cron and starts it. The caller has the responsibility to call the returned function to stop the cron jobs.
// All errors returned by a cron job or by etcd are logged using the logger in the context.
func Setup(ctx context.Context, opts SetupOpts) (func(), error) {
	log := logger.Get(ctx)

	if opts.EtcdClient != nil && opts.EtcdConfig != nil {
		return nil, errors.New(ctx, "both etcd client and config cannot be set")
	}

	cronOpts := []cron.Opt{
		cron.WithFuncCtx(funcCtx),
		cron.WithErrorsHandler(errorHandler),
		cron.WithEtcdErrorsHandler(errorHandler),
	}

	if opts.EtcdClient == nil && opts.EtcdConfig == nil {
		ctx, log = logger.WithFieldToCtx(ctx, "mode", "local")
	} else {
		ctx, log = logger.WithFieldToCtx(ctx, "mode", "distributed")

		etcdMutexBuilder, err := createEtcdMutextBuilderFromOpts(ctx, opts)
		if err != nil {
			return nil, errors.Wrap(ctx, err, "create the etcd mutex builder")
		}

		cronOpts = append(cronOpts, cron.WithEtcdMutexBuilder(etcdMutexBuilder))
	}

	c, err := cron.New(cronOpts...)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "create cron job runner")
	}

	var telemetry *telemetry
	if !opts.WithoutTelemetry {
		telemetry, err = newTelemetry(ctx)
		if err != nil {
			return nil, errors.Wrap(ctx, err, "init telemetry")
		}
	}

	for _, job := range opts.Jobs {
		if telemetry != nil {
			job = telemetry.wrapJob(job)
		}

		err := c.AddJob(job)
		if err != nil {
			return nil, errors.Wrap(ctx, err, "add the cron job")
		}
	}

	log.Info("Starting cron goroutine")

	c.Start(ctx)
	return func() {
		log.Info("Stopping cron goroutine")
		c.Stop()
	}, nil
}

func createEtcdMutextBuilderFromOpts(ctx context.Context, opts SetupOpts) (cron.EtcdMutexBuilder, error) {
	if opts.EtcdClient != nil {
		etcdMutexBuilder, err := cron.NewEtcdMutexBuilderFromClient(opts.EtcdClient)
		if err != nil {
			return nil, errors.Wrap(ctx, err, "create etcd mutex builder from client")
		}
		return etcdMutexBuilder, nil
	}

	etcdConfig, err := opts.EtcdConfig()
	if err != nil {
		return nil, errors.Wrap(ctx, err, "get etcd config")
	}

	etcdMutexBuilder, err := cron.NewEtcdMutexBuilder(etcdConfig)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "create etcd mutex builder from config")
	}

	return etcdMutexBuilder, nil
}

func funcCtx(ctx context.Context, j cron.Job) context.Context {
	log := logger.Get(ctx)
	requestID, ok := ctx.Value("request_id").(string)
	if !ok {
		requestUUID, err := uuid.NewV4()
		if err != nil {
			log.WithError(err).Error("Error generating UUID v4")
		} else {
			requestID = requestUUID.String()
			//nolint:revive,staticcheck // The "request_id" should not be of type string (https://pkg.go.dev/context#WithValue).
			// I don't know what would be the impact of using another type for such a field that is used in various repositories. Hence I'm disabling the linters.
			ctx = context.WithValue(ctx, "request_id", requestID)
		}
	}
	ctx, _ = logger.WithFieldsToCtx(ctx, logrus.Fields{
		"job_name":   j.Name,
		"request_id": requestID,
	})
	return ctx
}

func errorHandler(ctx context.Context, _ cron.Job, err error) {
	log := logger.Get(ctx)
	log.WithError(err).Error("Error when running cron job")
}
