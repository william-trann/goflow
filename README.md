# ForgeQ

> A distributed job queue for Go

ForgeQ is a layered job queue system built to look and feel like real backend infrastructure work. It models the pieces you would expect in a production-oriented queueing platform: durable queue semantics at the domain layer, a pluggable storage boundary, a worker runtime with configurable concurrency, explicit retry policy control, delayed scheduling, DLQ handling, queue statistics, and an HTTP control plane.

The current backend is in-memory by design. That keeps local development friction low while leaving clean seams for a Redis or database-backed adapter to be added without collapsing the architecture.

## Highlights

- Job enqueueing with queue-level routing
- Worker pool processing with configurable concurrency
- Explicit exponential backoff retry policy
- Priority-aware dequeue ordering
- Delayed and scheduled jobs
- Dead letter queue inspection and requeue
- Job status tracking across the full lifecycle
- Idempotency key support for safe enqueue retries
- Queue and system statistics
- Graceful runtime shutdown
- HTTP API plus a small Go client package
- Pluggable storage abstraction with an in-memory backend

## Installation

Install the combined runtime:

```bash
go install github.com/william-trann/goflow/cmd/forgeq@latest
```

Install the HTTP API binary:

```bash
go install github.com/william-trann/goflow/cmd/forgeq-api@latest
```

Install the worker binary:

```bash
go install github.com/william-trann/goflow/cmd/forgeq-worker@latest
```

Fetch the module:

```bash
go get github.com/william-trann/goflow
```

## Architecture

ForgeQ keeps domain logic away from transport and storage concerns.

```text
                 +----------------------+
                 |      HTTP API        |
                 | internal/api/http    |
                 +----------+-----------+
                            |
                 +----------v-----------+
                 |    Service Layer     |
                 | queue / dlq / metrics|
                 +----------+-----------+
                            |
                 +----------v-----------+
                 |   storage.Backend    |
                 |  pluggable contract  |
                 +----------+-----------+
                            |
      +---------------------+----------------------+
      |                                            |
+-----v------+                            +--------v--------+
| Worker Pool |                            | Scheduler Loop  |
| concurrency |                            | delayed promote |
+------------+                            +-----------------+
      |
+-----v-----------------------------------------------+
|               In-Memory Backend                     |
| ready heaps | delayed heap | idempotency index | dlq|
+-----------------------------------------------------+
```

## Queue Lifecycle

1. A client enqueues a job with queue, payload, priority, retry budget, optional schedule time, and optional idempotency key.
2. Immediate jobs enter a ready heap; future jobs enter the delayed scheduler heap.
3. The scheduler promotes due jobs into the ready heap.
4. Workers dequeue the highest-priority ready job for their configured queues.
5. Successful handlers ack the job and mark it `succeeded`.
6. Failed handlers either schedule a retry with exponential backoff or move the job to the DLQ once retry budget is exhausted.
7. DLQ jobs remain inspectable and can be manually requeued.

## Retry And DLQ Model

- `Attempts` increments when a job is claimed for processing.
- `MaxRetries` defines how many retry cycles are allowed after the first processing attempt.
- Retries use deterministic exponential backoff with a configurable base and max delay.
- Retryable failures move a job into `retrying` until the scheduler promotes it.
- Terminal failures move the job to `dead_lettered` with the last processor error attached.
- Requeueing from the DLQ resets attempts and places the job back into normal scheduling flow.

## HTTP API

Core endpoints:

- `POST /jobs`
- `GET /jobs/{id}`
- `GET /queues`
- `GET /queues/{name}/stats`
- `GET /dlq`
- `POST /dlq/{id}/requeue`

Example enqueue request:

```http
POST /jobs
Content-Type: application/json

{
  "queue": "emails",
  "payload": { "to": "ops@forgeq.dev", "template": "welcome" },
  "priority": 5,
  "max_retries": 3,
  "idempotency_key": "welcome-ops-001"
}
```

## Go Client Example

```go
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/william-trann/goflow/pkg/client"
)

func main() {
	c := client.New("http://localhost:8080", nil)

	job, err := c.Enqueue(context.Background(), client.EnqueueRequest{
		Queue:      "emails",
		Payload:    json.RawMessage(`{"to":"ops@forgeq.dev","template":"welcome"}`),
		Priority:   5,
		MaxRetries: 3,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("queued job %s with status %s", job.ID, job.Status)
}
```

## Examples

- `go run ./examples/basic`
- `go run ./examples/priority`
- `go run ./examples/retry`

## Repository Layout

```text
cmd/
  forgeq/
  forgeq-api/
  forgeq-worker/
examples/
  basic/
  priority/
  retry/
internal/
  api/
  config/
  core/
  dlq/
  errors/
  idempotency/
  metrics/
  model/
  queue/
  retry/
  scheduler/
  service/
  storage/
  worker/
pkg/
  client/
```

## Design Decisions

- Interface-driven storage keeps queue semantics independent from persistence details.
- Queue, DLQ, metrics, scheduling, and worker execution are split into focused packages with explicit boundaries.
- The worker runtime uses plain Go concurrency primitives instead of framework-style orchestration.
- The HTTP layer is intentionally thin and depends on services rather than storage directly.
- The in-memory adapter models realistic queue behavior instead of flattening everything into a map.
- The module path is set to `github.com/william-trann/goflow` so the project is ready to publish directly under that repository.

## Why This Project Exists

ForgeQ is designed as a serious portfolio-grade systems project. The goal is to show pragmatic backend engineering: clean package boundaries, credible runtime behavior, well-defined failure handling, transport isolation, and a codebase shape that could naturally evolve into a real open source queueing system.

## Local Development

Run the combined runtime:

```bash
make run
```

Run the API-only entrypoint:

```bash
make api
```

Run the worker-only entrypoint:

```bash
make worker
```

The combined runtime is the end-to-end path for the current in-memory backend. The split API and worker binaries mirror the intended deployment roles once a shared storage backend is introduced.

Configuration is driven by environment variables:

- `FORGEQ_HTTP_ADDR`
- `FORGEQ_WORKER_QUEUES`
- `FORGEQ_WORKER_CONCURRENCY`
- `FORGEQ_WORKER_POLL_INTERVAL`
- `FORGEQ_SCHEDULER_INTERVAL`
- `FORGEQ_RETRY_BASE_DELAY`
- `FORGEQ_RETRY_MAX_DELAY`

## Testing

Run the full test suite:

```bash
make test
```

The tests cover:

- priority dequeue ordering
- delayed job promotion
- exponential retry timing
- DLQ movement after retry exhaustion
- idempotent enqueue behavior

 persistence for idempotency records and DLQ retention policies
