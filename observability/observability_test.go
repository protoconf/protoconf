package observability

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// Init cannot be observed dialing an OTLP collector: otlptracegrpc.New builds its gRPC client
// non-blockingly, so even the enabled path never synchronously connects, and a listener-based
// "was I contacted" test would pass in both states and prove nothing. The sound observable proxy
// for "no exporter was constructed" is that a noop provider -- not an SDK provider -- got
// installed as the global provider. These tests assert on provider type identity instead.

func TestInit_disabled_installsNoopProviders(t *testing.T) {
	prevTracer := otel.GetTracerProvider()
	prevMeter := otel.GetMeterProvider()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTracer)
		otel.SetMeterProvider(prevMeter)
	})

	shutdown, err := Init(context.Background(), "test", false)
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	assert.IsType(t, tracenoop.NewTracerProvider(), otel.GetTracerProvider())
	assert.IsType(t, metricnoop.NewMeterProvider(), otel.GetMeterProvider())

	assert.NoError(t, shutdown(context.Background()))
}

func TestInit_enabled_installsSDKProviders(t *testing.T) {
	prevTracer := otel.GetTracerProvider()
	prevMeter := otel.GetMeterProvider()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTracer)
		otel.SetMeterProvider(prevMeter)
	})

	shutdown, err := Init(context.Background(), "test", true)
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	assert.IsType(t, &sdktrace.TracerProvider{}, otel.GetTracerProvider())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// With no collector listening, the meter provider's final flush is expected to fail; that
	// failure is irrelevant to what this test asserts (that the enabled path still installs real
	// SDK providers), so the shutdown error is deliberately ignored.
	_ = shutdown(ctx)
}
