package tracing

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	sentryotel "github.com/getsentry/sentry-go/otel"
	sentryotlp "github.com/getsentry/sentry-go/otel/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var tp *sdktrace.TracerProvider

// Enabled reports whether tracing is configured via environment variables.
// Set SENTRY_DSN and/or OTEL_EXPORTER_OTLP_ENDPOINT to enable.
func Enabled() bool {
	return os.Getenv("SENTRY_DSN") != "" || os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != ""
}

// Init initializes the OpenTelemetry TracerProvider. It configures exporters
// based on environment variables:
//
//   - SENTRY_DSN: exports traces to Sentry via its OTel integration
//   - SENTRY_TRACES_SAMPLE_RATE: sample rate for Sentry (default: 1.0)
//   - OTEL_EXPORTER_OTLP_ENDPOINT: exports traces to an OTLP-compatible collector
//   - OTEL_EXPORTER_OTLP_HEADERS: headers for the OTLP exporter (standard OTel env var)
//
// Both exporters can be active simultaneously. If neither env var is set, Init is a no-op.
func Init(ctx context.Context) error {
	if !Enabled() {
		return nil
	}

	sentryDSN := os.Getenv("SENTRY_DSN")
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("searxng-mcp"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return fmt.Errorf("creating otel resource: %w", err)
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
	}

	propagators := []propagation.TextMapPropagator{
		propagation.TraceContext{},
		propagation.Baggage{},
	}

	if sentryDSN != "" {
		sampleRate := 1.0
		if v := os.Getenv("SENTRY_TRACES_SAMPLE_RATE"); v != "" {
			if parsed, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
				sampleRate = parsed
			}
		}

		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentryDSN,
			EnableTracing:    true,
			TracesSampleRate: sampleRate,
			Integrations: func(integrations []sentry.Integration) []sentry.Integration {
				return append(integrations, sentryotel.NewOtelIntegration())
			},
		}); err != nil {
			return fmt.Errorf("sentry init: %w", err)
		}

		exporter, err := sentryotlp.NewTraceExporter(ctx, sentryDSN)
		if err != nil {
			return fmt.Errorf("sentryotlp.NewTraceExporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	if otlpEndpoint != "" {
		exporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return fmt.Errorf("otlp exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	tp = sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagators...))

	return nil
}

// Shutdown flushes pending spans and shuts down the TracerProvider.
func Shutdown(ctx context.Context) error {
	var firstErr error

	if tp != nil {
		if err := tp.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if os.Getenv("SENTRY_DSN") != "" {
		sentry.Flush(2 * time.Second)
	}

	return firstErr
}
