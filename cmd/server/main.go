package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tilshansanoj/open-telemetry-demo/internal/handler"
	"github.com/tilshansanoj/open-telemetry-demo/internal/telemetry"
)

func main() {
	collectorEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4317")
	serviceName := getEnv("SERVICE_NAME", "otel-demo-app")
	port := getEnv("PORT", "8080")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tp, err := telemetry.InitTracerProvider(ctx, serviceName, collectorEndpoint)
	if err != nil {
		log.Fatalf("failed to init tracer provider: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handler.HandleRoot)
	mux.HandleFunc("GET /ping", handler.HandlePing)
	mux.HandleFunc("GET /work", handler.HandleWork)

	wrapped := telemetry.NewMiddleware(serviceName)(mux)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: wrapped,
	}

	go func() {
		log.Printf("server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	if err := tp.Shutdown(shutdownCtx); err != nil {
		log.Printf("tracer provider shutdown error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
