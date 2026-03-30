# OpenTelemetry Go Demo

An end-to-end distributed tracing demo using the OpenTelemetry Go SDK. A Go HTTP service emits traces that flow through an OTel Collector into Grafana Tempo, then visualised in Grafana dashboards — all wired together with Docker Compose.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Docker Network                        │
│                                                              │
│  ┌──────────┐   OTLP gRPC    ┌───────────────┐   OTLP gRPC  │
│  │  Go App  │ ─────────────► │ OTel Collector│ ────────────►│
│  │  :8080   │                │  :4317 :4318  │              │
│  └──────────┘                └───────────────┘              │
│                                      │                       │
│                                      ▼                       │
│                              ┌──────────────┐               │
│                              │ Grafana Tempo│               │
│                              │    :3200     │               │
│                              └──────────────┘               │
│                                      │                       │
│                                      ▼                       │
│                              ┌──────────────┐               │
│                              │   Grafana    │               │
│                              │    :3000     │               │
│                              └──────────────┘               │
└─────────────────────────────────────────────────────────────┘
```

### Data flow

1. An HTTP request arrives at the Go app on `:8080`
2. The `otelhttp` middleware automatically creates a **server span** for every request, extracting W3C `traceparent` headers for distributed context propagation
3. The `HandleWork` handler creates a child **`HandleWork` span**, and `doWork` creates a grandchild **`doWork` span** with a `work.sleep.ms=50` attribute
4. Spans are batched and exported via **OTLP gRPC** to the OTel Collector on `otel-collector:4317`
5. The collector processes spans through a batch processor, then forwards them to **Grafana Tempo** and logs them via the debug exporter
6. Tempo stores traces locally and exposes a query API on `:3200`
7. **Grafana** queries Tempo via a pre-provisioned datasource — browse traces in the Explore UI

---

## Repository Structure

```
.
├── cmd/
│   └── server/
│       └── main.go                     # Entry point: server bootstrap, signal handling
├── internal/
│   ├── handler/
│   │   └── handler.go                  # HTTP handlers with manual span instrumentation
│   └── telemetry/
│       └── provider.go                 # TracerProvider setup and HTTP middleware
├── grafana/
│   └── datasources/
│       └── tempo.yaml                  # Auto-provisioned Tempo datasource for Grafana
├── docker-compose.yml                  # Full observability stack
├── Dockerfile                          # Multi-stage Go build
├── otel-collector-config.yaml          # Collector pipeline: receive → batch → export
├── tempo.yaml                          # Tempo trace storage config
└── go.mod                              # Go module: github.com/tilshansanoj/open-telemetry-demo
```

---

## Code Walkthrough

### `cmd/server/main.go` — Entry Point

Reads configuration from environment variables, initialises the OTel `TracerProvider`, registers HTTP routes, starts the server, and handles graceful shutdown.

```
OTEL_EXPORTER_OTLP_ENDPOINT  (default: collector:4317)
SERVICE_NAME                  (default: otel-demo-app)
PORT                          (default: 8080)
```

Shutdown order is critical: the HTTP server stops accepting requests first, then `tp.Shutdown()` is called to flush any buffered spans before the process exits. Skipping this would silently drop the last batch of traces.

### `internal/telemetry/provider.go` — OTel Bootstrap

**`InitTracerProvider`** wires up the full OTel SDK pipeline:

| Component | What it does |
|---|---|
| `otlptracegrpc.New` | Opens a gRPC connection to the collector; `WithInsecure()` disables TLS (fine for local demo) |
| `resource.New` | Attaches `service.name` to every span so Tempo and Grafana can filter by service |
| `sdktrace.WithBatcher` | Buffers spans and sends them in batches — more efficient than sending one-by-one |
| `otel.SetTracerProvider` | Registers the provider globally so `otel.Tracer(...)` works anywhere in the codebase |
| `propagation.TraceContext{}` | Enables W3C `traceparent` header parsing for distributed tracing across services |

**`NewMiddleware`** wraps the HTTP mux with `otelhttp.NewHandler`, which automatically:
- Creates a server span for every incoming request
- Records HTTP method, route, and status code as span attributes
- Extracts incoming `traceparent` headers to continue a trace from an upstream caller

### `internal/handler/handler.go` — HTTP Handlers

Three endpoints, each producing different span shapes:

| Endpoint | Handler | Spans produced |
|---|---|---|
| `GET /` | `HandleRoot` | 1 server span (from middleware) |
| `GET /ping` | `HandlePing` | 1 server span (from middleware) |
| `GET /work` | `HandleWork` | 1 server span + `HandleWork` child + `doWork` grandchild |

The `GET /work` span tree demonstrates **context propagation** — each function receives a `context.Context` containing the active span. Calling `otel.Tracer("handler").Start(ctx, "doWork")` creates a child of whatever span is in `ctx`, linking them into a single trace automatically.

The `doWork` span also records a custom attribute:
```go
span.SetAttributes(attribute.Int("work.sleep.ms", 50))
```
This is visible in the Grafana Tempo span detail view.

---

## Infrastructure

### `docker-compose.yml` — Service Stack

| Service | Image | Ports | Role |
|---|---|---|---|
| `app` | Built from `Dockerfile` | `8080` | Go HTTP service |
| `otel-collector` | `otel/opentelemetry-collector-contrib:latest` | `4317` (gRPC), `4318` (HTTP) | Receives, batches, and forwards spans |
| `tempo` | `grafana/tempo:2.6.1` | `3200` | Stores and queries traces |
| `grafana` | `grafana/grafana:latest` | `3000` | Dashboard and trace explorer UI |

Startup order: `tempo` starts first, then `otel-collector` (depends on `tempo`), then `app` (depends on `otel-collector`). Grafana starts independently alongside tempo.

### `otel-collector-config.yaml` — Collector Pipeline

```
receivers:  otlp (gRPC :4317, HTTP :4318)
     │
processors: batch  ← buffers spans to reduce export calls
     │
exporters:  otlp → tempo:4317   ← sends to Tempo
            debug               ← logs span details to collector stdout
```

### `tempo.yaml` — Trace Backend

Tempo receives OTLP spans from the collector on its internal gRPC port `4317`. Traces are stored on local disk under `/tmp/tempo/` (ephemeral — cleared on container restart). The HTTP API on `:3200` is what Grafana queries.

### `Dockerfile` — Multi-Stage Build

```
Stage 1 (builder): golang:1.25-alpine
  → go mod download  (cached layer)
  → go build -o /app ./cmd/server

Stage 2 (runtime): alpine:3.19
  → copies /app binary only (~10MB image)
  → no Go toolchain in the final image
```

---

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Compose v2 (`docker compose` command)
- Go 1.22+ (only needed for local development without Docker)

---

## Setup and Running

### 1. Clone the repository

```bash
git clone https://github.com/tilshansanoj/open-telemetry-demo
cd open-telemetry-demo
```

### 2. Start the full stack

```bash
docker compose up -d --build
```

This builds the Go app image and starts all four services. First run downloads images and Go dependencies — allow ~2 minutes.

### 3. Verify services are up

```bash
docker compose ps
```

All four services should show `running`.

### 4. Generate trace data

```bash
curl http://localhost:8080/ping
curl http://localhost:8080/
curl http://localhost:8080/work
```

### 5. Explore traces in Grafana

1. Open [http://localhost:3000](http://localhost:3000) in your browser (no login required — anonymous access enabled)
2. Click **Explore** in the left sidebar
3. Select **Tempo** from the datasource dropdown
4. Click **Search** and then **Run query**
5. Click any trace to expand the span tree

### 6. Query Tempo directly (optional)

```bash
# List recent traces
curl "http://localhost:3200/api/search?limit=10"

# Fetch a specific trace by ID
curl "http://localhost:3200/api/traces/<traceID>"
```

### 7. View collector debug output

```bash
docker compose logs otel-collector -f
```

The debug exporter prints every span received, useful for confirming spans are flowing before checking Tempo.

---

## Local Development (without Docker)

To run the Go service locally while pointing at a separately-running collector:

```bash
# Install dependencies
go mod download

# Run the server (expects collector on localhost:4317)
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
SERVICE_NAME=otel-demo-app \
PORT=8080 \
go run ./cmd/server
```

To build the binary:

```bash
go build -o bin/server ./cmd/server
./bin/server
```

---

## Stopping the Stack

```bash
docker compose down
```

Add `-v` to also remove any named volumes if you add persistent storage later.

---

## Key Concepts

**Trace** — a tree of spans representing a single request as it travels through your system. Each trace has a unique `traceID`.

**Span** — a single unit of work within a trace. Has a name, start/end timestamps, a `spanID`, a parent `spanID` (except the root), and optional attributes.

**Context propagation** — passing the active span's `traceID` and `spanID` through function calls (via `context.Context` in Go) and across network boundaries (via `traceparent` HTTP header). This is what links spans into a tree.

**OTLP** — OpenTelemetry Protocol. The standard wire format for exporting telemetry data. This demo uses OTLP over gRPC.

**OTel Collector** — a vendor-neutral proxy that receives telemetry, processes it (batching, filtering, enriching), and fans it out to one or more backends. Decouples the app from the storage backend.

**W3C TraceContext** — the standard HTTP header format (`traceparent`) for propagating trace context between services. Enables distributed tracing across microservices.
