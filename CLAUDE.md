# OTel Go Demo

## Project context
End-to-end OpenTelemetry tracing demo using Go. Targets:
- Go service with OTel SDK (traces + metrics)
- OTel Collector via Docker Compose
- Grafana Tempo as trace backend
- Grafana for dashboards

## Tech rules
- Go 1.22+, module path: github.com/tilshansanoj/open-telemetry-demo
- OTel SDK: go.opentelemetry.io/otel v1.x
- Exporter: OTLP gRPC to collector:4317
- Docker Compose v2 syntax

## Agent routing rules
- Scaffold Go structure → use go-scaffolder subagent
- Add instrumentation to existing Go files → use otel-instrumentor subagent
- Docker/infra config → use infra-builder subagent
- Verify traces end-to-end → use trace-verifier subagent