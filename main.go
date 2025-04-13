package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

var (
	requestCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hello_requests_total",
			Help: "Total number of requests to the hello endpoint",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hello_request_duration_seconds",
			Help:    "Duration of hello requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

func main() {
	// Add metrics endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2222", nil)
	}()

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	shutdown := initTracer()
	defer shutdown()

	r := gin.Default()

	r.Use(otelgin.Middleware("hello-service"))

	r.GET("/hello", func(c *gin.Context) {
		start := time.Now()

		tr := otel.Tracer("hello-service")
		_, span := tr.Start(c.Request.Context(), "say-hello")
		defer span.End()

		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()

		logrus.WithFields(logrus.Fields{
			"trace_id": traceID,
			"span_id":  spanID,
			"path":     c.Request.URL.Path,
			"method":   c.Request.Method,
		}).Info("Handled /hello request")

		c.String(http.StatusOK, "Hello, traced Gin")

		// Record metrics
		status := strconv.Itoa(http.StatusOK)
		requestCount.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Inc()
		requestDuration.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).
			Observe(time.Since(start).Seconds())
	})

	log.Println("Listening on http://localhost:8080")
	r.Run(":8080")
}

func initTracer() func() {
	ctx := context.Background()

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("jaeger:4318"),
		otlptracehttp.WithInsecure(),
	)

	if err != nil {
		log.Fatalf("failed to create OTLP exporter: %v", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName("hello-service"))),
	)

	otel.SetTracerProvider(tp)

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatalf("Error shutting down tracer provider: %v", err)
		}
	}
}
