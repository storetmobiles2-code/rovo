// Package obs wires observability: slog JSON logging with request/trace ids and PII redaction,
// an OTel tracer that exports over OTLP when configured, and a Prometheus registry with HTTP
// metrics (24, charter §2).
package obs

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	loggerKey
)

// NewLogger builds the slog JSON logger. The ReplaceAttr hook redacts values under PII-ish keys
// defensively; structured logging should never carry raw PII in the first place (19).
func NewLogger(level slog.Level) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: redactAttr,
	})
	return slog.New(h)
}

var piiKeys = map[string]bool{
	"phone": true, "phone_e164": true, "email": true, "password": true, "otp": true,
	"code": true, "token": true, "authorization": true, "address": true, "full_name": true,
	"pan": true, "vpa": true, "account_number": true, "dl_number": true, "totp": true,
}

func redactAttr(_ []string, a slog.Attr) slog.Attr {
	if piiKeys[a.Key] {
		return slog.String(a.Key, "[redacted]")
	}
	return a
}

// WithRequestID returns ctx carrying a request id and a logger bound to it (plus trace ids when
// a span is recording).
func WithRequestID(ctx context.Context, logger *slog.Logger, requestID string) context.Context {
	l := logger.With(slog.String("request_id", requestID))
	if sc := oteltrace.SpanContextFromContext(ctx); sc.IsValid() {
		l = l.With(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	ctx = context.WithValue(ctx, requestIDKey, requestID)
	return context.WithValue(ctx, loggerKey, l)
}

// RequestID returns the request id in ctx, or "".
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// L returns the context logger, or base.
func L(ctx context.Context, base *slog.Logger) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return base
}

// TraceID returns the current trace id, or "".
func TraceID(ctx context.Context) string {
	if sc := oteltrace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

// Tracing holds the tracer provider for shutdown.
type Tracing struct {
	tp *sdktrace.TracerProvider
}

// SetupTracing installs an OTLP tracer provider when endpoint != "", else a no-op. The W3C
// traceparent propagator is always set so trace ids flow through River job args and SSE.
func SetupTracing(ctx context.Context, endpoint, serviceName, env string) (*Tracing, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	if endpoint == "" {
		return &Tracing{}, nil
	}
	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	if err != nil {
		return nil, err
	}
	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL,
		semconv.ServiceName(serviceName), attribute.String("deployment.environment", env)))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	return &Tracing{tp: tp}, nil
}

// Shutdown flushes pending spans.
func (t *Tracing) Shutdown(ctx context.Context) error {
	if t == nil || t.tp == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return t.tp.Shutdown(ctx)
}

// Metrics holds the Prometheus registry and the standard HTTP metrics.
type Metrics struct {
	Registry      *prometheus.Registry
	RequestsTotal *prometheus.CounterVec
	Duration      *prometheus.HistogramVec
	InFlight      prometheus.Gauge
	SSEActive     prometheus.Gauge
}

// NewMetrics registers Go runtime collectors and rovo's HTTP metrics.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	m := &Metrics{
		Registry: reg,
		RequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rovo_http_requests_total", Help: "HTTP requests by route, method and status.",
		}, []string{"route", "method", "status"}),
		Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "rovo_http_request_duration_seconds", Help: "HTTP request duration.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		}, []string{"route", "method"}),
		InFlight:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "rovo_http_in_flight_requests", Help: "In-flight HTTP requests."}),
		SSEActive: prometheus.NewGauge(prometheus.GaugeOpts{Name: "rovo_sse_active_streams", Help: "Open SSE streams on this replica."}),
	}
	reg.MustRegister(m.RequestsTotal, m.Duration, m.InFlight, m.SSEActive)
	return m
}

// ObserveRequest records one HTTP request.
func (m *Metrics) ObserveRequest(route, method string, status int, d time.Duration) {
	if m == nil {
		return
	}
	m.RequestsTotal.WithLabelValues(route, method, statusClass(status)).Inc()
	m.Duration.WithLabelValues(route, method).Observe(d.Seconds())
}

func statusClass(s int) string {
	switch {
	case s >= 500:
		return "5xx"
	case s >= 400:
		return "4xx"
	case s >= 300:
		return "3xx"
	case s >= 200:
		return "2xx"
	default:
		return "1xx"
	}
}
