// Package telemetry initializes OpenTelemetry tracing, metrics and the slog JSON logger.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/n1ckerr0r/avito-labs/internal/config"
)

// Providers bundles the tracer and meter providers the rest of the service
// needs, plus a single shutdown hook for both.
type Providers struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Shutdown       func(context.Context) error
}

// Setup builds the OTel SDK from cfg. Exporter creation does not dial the
// collector, so a collector that is down at startup does not fail this call.
func Setup(ctx context.Context, cfg config.Config) (*Providers, error) {
	res, err := resource.New(ctx, resource.WithFromEnv(), resource.WithAttributes(semconv.ServiceVersion("dev")))
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	sampler, err := newSampler(cfg.OTELTracesSampler)
	if err != nil {
		return nil, err
	}

	traceExporter, err := newTraceExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("build trace exporter: %w", err)
	}

	metricExporter, err := newMetricExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("build metric exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))),
		sdkmetric.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	shutdown := func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx))
	}

	return &Providers{TracerProvider: tp, MeterProvider: mp, Shutdown: shutdown}, nil
}

// splitEndpoint turns a full URL (as OTEL_EXPORTER_OTLP_ENDPOINT provides it)
// into a bare host:port and an insecure flag. The OTLP exporters' WithEndpoint
// (unlike WithEndpointURL) appends the signal-specific path ("/v1/traces",
// "/v1/metrics") automatically, which is what OTEL_EXPORTER_OTLP_ENDPOINT
// being signal-agnostic requires.
func splitEndpoint(endpoint string) (host string, insecure bool, err error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", false, fmt.Errorf("parse OTEL_EXPORTER_OTLP_ENDPOINT: %w", err)
	}

	return u.Host, u.Scheme != "https", nil
}

func newTraceExporter(ctx context.Context, cfg config.Config) (sdktrace.SpanExporter, error) {
	host, insecure, err := splitEndpoint(cfg.OTELExporterEndpoint)
	if err != nil {
		return nil, err
	}

	if cfg.OTELExporterProtocol == "grpc" {
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(host)}
		if insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}

		return otlptracegrpc.New(ctx, opts...)
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(host)}
	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	return otlptracehttp.New(ctx, opts...)
}

func newMetricExporter(ctx context.Context, cfg config.Config) (sdkmetric.Exporter, error) {
	host, insecure, err := splitEndpoint(cfg.OTELExporterEndpoint)
	if err != nil {
		return nil, err
	}

	if cfg.OTELExporterProtocol == "grpc" {
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(host)}
		if insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}

		return otlpmetricgrpc.New(ctx, opts...)
	}

	opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(host)}
	if insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	return otlpmetrichttp.New(ctx, opts...)
}

func newSampler(name string) (sdktrace.Sampler, error) {
	switch name {
	case "always_on":
		return sdktrace.AlwaysSample(), nil
	case "always_off":
		return sdktrace.NeverSample(), nil
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample()), nil
	default:
		return nil, fmt.Errorf("unsupported OTEL_TRACES_SAMPLER %q", name)
	}
}
