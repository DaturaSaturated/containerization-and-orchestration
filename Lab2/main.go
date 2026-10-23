package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of http requests",
	}, []string{"endpoint", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of http requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"endpoint"})
)

func main() {
	ctx := context.Background()
	
	// Инициализация OpenTelemetry (эндпоинт берется из OTEL_EXPORTER_OTLP_ENDPOINT)
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		slog.Error("failed to create otel exporter", "err", err)
	}
	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("lab2-api"),
		),
	)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(ctx) }()

	tracer := otel.Tracer("lab2-api")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mux := http.NewServeMux()

	// Middleware для трейсинга, логирования и метрик
	handler := func(next http.HandlerFunc, endpoint string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx, span := tracer.Start(r.Context(), endpoint)
			defer span.End()

			traceID := span.SpanContext().TraceID().String()
			reqLogger := logger.With("trace_id", traceID, "endpoint", endpoint)

			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next(wrapped, r.WithContext(ctx))

			duration := time.Since(start).Seconds()
			status := fmt.Sprintf("%d", wrapped.statusCode)
			
			requestsTotal.WithLabelValues(endpoint, status).Inc()
			requestDuration.WithLabelValues(endpoint).Observe(duration)

			reqLogger.Info("request processed", "status", wrapped.statusCode, "duration", duration)
			
			if wrapped.statusCode >= 500 {
				span.SetStatus(codes.Error, "internal server error")
			}
		}
	}

	mux.HandleFunc("/health", handler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}, "/health"))

	mux.HandleFunc("/fail", handler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}, "/fail"))

	mux.HandleFunc("/slow", handler(func(w http.ResponseWriter, r *http.Request) {
		_, span := tracer.Start(r.Context(), "slow-op")
		defer span.End()
		
		sleepTime := time.Duration(1000+rand.Intn(2000)) * time.Millisecond
		time.Sleep(sleepTime)
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("slept for %s", sleepTime)))
	}, "/slow"))

	mux.HandleFunc("/load", handler(func(w http.ResponseWriter, r *http.Request) {
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequestWithContext(r.Context(), "GET", "http://localhost:8080/health", nil)
				resp, err := http.DefaultClient.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}()
		}
		wg.Wait()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("load generated"))
	}, "/load"))

	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		slog.Info("shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	slog.Info("starting server on :8080")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server error", "err", err)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}