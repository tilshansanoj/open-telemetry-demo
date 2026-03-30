package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func HandlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func HandleWork(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("handler").Start(r.Context(), "HandleWork")
	defer span.End()

	doWork(ctx)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": "done"})
}

func doWork(ctx context.Context) {
	_, span := otel.Tracer("handler").Start(ctx, "doWork")
	defer span.End()

	span.SetAttributes(attribute.Int("work.sleep.ms", 50))
	time.Sleep(50 * time.Millisecond)
}
