package logger

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
)

// Opt is a function-option type for the Default() method.
type Opt func(*logrus.Logger)

// WithLogLevel let us define the level of verbosity of the logger
func WithLogLevel(lvl logrus.Level) Opt {
	return func(l *logrus.Logger) {
		l.SetLevel(lvl)
	}
}

func WithLogFormatter(f logrus.Formatter) Opt {
	return func(l *logrus.Logger) {
		l.SetFormatter(f)
	}
}

func WithHooks(hooks []logrus.Hook) Opt {
	return func(l *logrus.Logger) {
		for _, h := range hooks {
			l.Hooks.Add(h)
		}
	}
}

// Default generate a logrus logger with the configuration defined in the environment and the hooks used in the plugins
func Default(opts ...Opt) logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(logLevel())
	logger.Formatter = formatter()

	for _, hook := range Plugins().Hooks() {
		logger.Hooks.Add(hook)
	}

	for _, opt := range opts {
		opt(logger)
	}

	var fieldLogger logrus.FieldLogger = logger
	if os.Getenv("REGION_NAME") != "" {
		fieldLogger = fieldLogger.WithField("region", os.Getenv("REGION_NAME"))
	}

	return fieldLogger
}

// NewContextWithLogger generate a new context (based on context.Background()) and add a Default() logger on top of it
func NewContextWithLogger() context.Context {
	return AddLoggerToContext(context.Background())
}

// AddLoggerToContext add the Default() logger on top of the current context
func AddLoggerToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, "logger", Default())
}

// Get return the logger stored in the context or create a new one if the logger is not set
func Get(ctx context.Context) logrus.FieldLogger {
	if logger, ok := ctx.Value("logger").(logrus.FieldLogger); ok {
		return logger
	}

	return Default().WithField("invalid_context", true)
}

// ToCtx add a logger to a context
func ToCtx(ctx context.Context, logger logrus.FieldLogger) context.Context {
	return context.WithValue(ctx, "logger", logger)
}
