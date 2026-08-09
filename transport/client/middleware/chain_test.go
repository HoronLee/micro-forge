package middleware

import (
	"net/http"
	"testing"

	corev1 "github.com/Servora-Kit/servora/api/gen/go/servora/core/v1"
	"github.com/Servora-Kit/servora/obs/metrics"
	"go.opentelemetry.io/otel/metric/noop"
	"log/slog"
)

func createTestMetrics() *metrics.Metrics {
	meter := noop.NewMeterProvider().Meter("test")
	requests, _ := meter.Int64Counter("test_requests")
	seconds, _ := meter.Float64Histogram("test_seconds")
	return &metrics.Metrics{
		ClientRequests: requests,
		ClientSeconds:  seconds,
		Handler:        http.NotFoundHandler(),
	}
}

func TestNewChainBuilder_BasicBuild(t *testing.T) {
	ms := NewChainBuilder(slog.Default()).Build()
	if len(ms) != 3 {
		t.Errorf("expected 3 middlewares (recovery,logging,circuit), got %d", len(ms))
	}
}

func TestChainBuilder_WithTrace_Enabled(t *testing.T) {
	trace := &corev1.Trace{Endpoint: "http://otel:4317"}
	ms := NewChainBuilder(slog.Default()).WithTrace(trace).Build()
	if len(ms) != 4 {
		t.Errorf("expected 4 middlewares with tracing, got %d", len(ms))
	}
}

func TestChainBuilder_WithMetrics_Enabled(t *testing.T) {
	mtc := createTestMetrics()
	ms := NewChainBuilder(slog.Default()).WithMetrics(mtc).Build()
	if len(ms) != 4 {
		t.Errorf("expected 4 middlewares with metrics, got %d", len(ms))
	}
}

func TestChainBuilder_WithoutCircuitBreaker(t *testing.T) {
	ms := NewChainBuilder(slog.Default()).WithoutCircuitBreaker().Build()
	if len(ms) != 2 {
		t.Errorf("expected 2 middlewares without circuitbreaker, got %d", len(ms))
	}
}
