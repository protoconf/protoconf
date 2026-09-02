package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/protoconf/protoconf/consts"
)

// Init initializes OpenTelemetry trace and metric providers with OTLP gRPC exporters.
// If enabled is false, no exporter is constructed and no resource detection is performed:
// noop trace and meter providers are installed explicitly and Init returns a nil error
// alongside a working no-op shutdown function -- being switched off is not a failure.
// If the trace exporter is unavailable, noop providers are installed and a non-nil error
// is returned alongside a no-op shutdown function.
// If the metric exporter is unavailable, a noop meter provider is installed and a partial
// shutdown function (trace only) is returned with the error.
// Init always returns a non-nil shutdown function.
func Init(ctx context.Context, serviceName string, enabled bool) (func(context.Context) error, error) {
	if !enabled {
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
		otel.SetMeterProvider(metricnoop.NewMeterProvider())
		return func(context.Context) error { return nil }, nil
	}

	resources, _ := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(consts.Version),
		),
	)

	expTracer, err := otlptracegrpc.New(ctx)
	if err != nil {
		slog.Warn("OTel trace exporter unavailable, using noop", "error", err)
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
		otel.SetMeterProvider(metricnoop.NewMeterProvider())
		return func(context.Context) error { return nil }, fmt.Errorf("trace exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(expTracer),
		sdktrace.WithResource(resources),
	)
	otel.SetTracerProvider(tracerProvider)

	expMeter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		slog.Warn("OTel metric exporter unavailable, using noop", "error", err)
		otel.SetMeterProvider(metricnoop.NewMeterProvider())
		return func(ctx context.Context) error {
			return tracerProvider.Shutdown(ctx)
		}, fmt.Errorf("metric exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(expMeter)),
		sdkmetric.WithResource(resources),
	)
	otel.SetMeterProvider(meterProvider)

	return func(ctx context.Context) error {
		return errors.Join(
			tracerProvider.Shutdown(ctx),
			meterProvider.Shutdown(ctx),
		)
	}, nil
}
