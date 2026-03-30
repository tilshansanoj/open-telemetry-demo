---
name: infra-builder
description: Creates Docker Compose infrastructure for OTel Collector, Tempo, and Grafana. Use when setting up or modifying observability backend.
tools: Read, Write, Edit, Glob
---

You create Docker Compose observability stacks.
Services to configure:
- otel-collector (otel/opentelemetry-collector-contrib) listening on :4317 gRPC, :4318 HTTP
- tempo (grafana/tempo) receiving OTLP from collector
- grafana with Tempo datasource pre-configured

Output: docker-compose.yml, otel-collector-config.yaml, tempo.yaml, grafana/datasources/tempo.yaml
```

**Step 4 — Invoke agents from Claude Code**

Claude can spawn subagents — you can define user-designed subagents and tell Claude exactly when and how to use them.  Just talk naturally:
```
# Start Claude Code in your project
claude

# Then just prompt:
"Scaffold the Go service first using the go-scaffolder agent,
then use otel-instrumentor to add tracing to all HTTP handlers"

# Or trigger sequentially:
"Use go-scaffolder to create the service, then
use infra-builder to create the Docker Compose stack in parallel"
```

You can also explicitly request a subagent by name — for example, "Use the go-scaffolder agent to create the main.go". This bypasses automatic matching and directly invokes the named subagent. 

**Step 5 — Context window tips**

Each subagent has its own context window, which means they can execute a series of searches or code reads without any of that content accumulating in your main conversation. The parent receives a concise summary, not every file the subagent read. 

The rule of thumb: if a task is going to read 10+ files or generate a lot of output you don't need to see, delegate it to a subagent.

**Step 6 — Your overall session flow**
```
1. claude                           ← start Claude Code in your project dir
2. "Read CLAUDE.md and plan the OTel demo build"
3. "Use go-scaffolder to scaffold the service"
4. "Use otel-instrumentor to add tracing"
5. "Use infra-builder to create docker-compose.yml and configs"
6. "Start the stack with docker compose up -d"
7. "Use trace-verifier to send test requests and confirm traces appear in Tempo"