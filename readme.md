# Go Worker + SNS/SQS + DynamoDB Learning Plan

## Goal

Build a small Go application step by step to learn and internalize:

- worker pools
- cancellation with `context`
- fail-fast coordination with `errgroup`
- clean channel ownership and shutdown
- SQS polling
- SNS fan-out
- multiple job types
- persisting results to DynamoDB

---

# Phase 0 — Project setup [x]

Create the basic project structure:

- initialize git repo
- initialize Go module
- create `main.go`
- use a clean `main()` + `run(ctx)` lifecycle pattern
- add signal-aware shutdown with `signal.NotifyContext`

**Outcome:** a clean, idiomatic Go application entrypoint.

---

# Phase 1 — Local in-process worker pool [x]

Build the core concurrency model without AWS:

- create a `jobs` channel
- create a `results` channel
- define a simple `Job` and `Result`
- start a fixed number of workers
- create a simple local producer that sends test jobs
- create a consumer that ranges over results and prints them
- close channels with correct ownership rules

**Outcome:** a working local worker-pool demo with bounded concurrency.

---

# Phase 2 — Add cancellation and fail-fast behavior [x]

Enhance the worker pool with:

- `context.Context`
- `errgroup.WithContext`
- early cancellation on fatal worker errors
- safe worker exit paths
- cancellable result sends
- first-error return semantics

**Outcome:** the worker pool can stop cleanly and return a meaningful fatal error.

---

# Phase 3 — Add graceful shutdown [x]

Make the app behave like a real long-running process:

- root context cancelled by OS signals
- workers and producers stop cleanly on shutdown
- no blocked goroutines
- no leaked goroutines
- no panics on exit

**Outcome:** Ctrl+C or SIGTERM causes a clean shutdown.

---

# Phase 4 — Replace local producer with SQS polling [x]

Swap out the fake in-process producer for a real poller:

- long-poll SQS
- convert SQS messages into `Job`
- push jobs into the local worker pool
- respect cancellation and backpressure

**Outcome:** the app becomes a real queue-driven worker service.

---

# Phase 5 — Add SNS → SQS fan-out [x]

Introduce AWS event fan-out:

- create an SNS topic
- subscribe one or more SQS queues
- publish to SNS
- confirm SNS delivers messages into SQS
- let the worker service consume from SQS

**Outcome:** understand and demonstrate the common SNS→SQS event pattern.

---

# Phase 6 — Add multiple job types [x]

Expand the system to handle multiple kinds of work:

- add a `Type` field to `Job`
- support at least two types, e.g. string jobs and int jobs
- route processing logic based on job type

**Outcome:** one worker system can process multiple categories of jobs.

---

# Phase 7 — Split fan-out across multiple queues - BUMP TO END

Make SNS fan-out more realistic:

- use one SNS topic and multiple SQS subscriptions
- optionally filter different message types into different queues
- poll multiple queues
- feed different job streams into the worker system

**Outcome:** model a more realistic event-driven architecture with independent consumers.

---

# Phase 8 — Persist results to DynamoDB

Replace simple console printing with storage:

- create a DynamoDB table
- define a result persistence shape
- write processed job results into DynamoDB
- keep result handling separate from worker logic

**Outcome:** processed results are stored in a real NoSQL database.

---

# Phase 9 — Add observability

Improve visibility into the system:

- add structured logging
- optionally add a separate event/log stream for non-fatal chatter
- keep fatal errors separate from logs
- make shutdown and processing visible in logs

**Outcome:** the system becomes easier to observe and debug.

---

# Phase 10 — Optional service wrapper

Only if useful later, add a lightweight service wrapper:

- health endpoint
- readiness endpoint
- optional metrics endpoint
- optional debug/publish endpoint

This is optional and should come after the worker system itself is stable.

**Outcome:** the worker can run as a more production-like service process.

---

# Recommended order

1. Project setup
2. Local worker pool
3. Cancellation + errgroup
4. Graceful shutdown
5. SQS polling
6. SNS fan-out
7. Multiple job types
8. DynamoDB persistence
9. Observability
10. Optional service wrapper

---

# Final principle

Build it in layers:

1. local concurrency demo
2. long-running worker process
3. AWS-backed event pipeline

Do not over-abstract early.
Do not start with HTTP.
Do not introduce interfaces too soon.

Start small, get it working, then evolve it.

# Localstack
AWS CLI wrapper, awslocal, which can be configured to point to local docker env.

## SQS
```
awslocal sqs create-queue --queue-name test-queue
```
```
awslocal sqs list-queues
```
```
awslocal sqs send-message \
  --queue-url http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/test-queue \
  --message-body '{"type":"int","value":5}'
```
```
awslocal sqs receive-message \
  --queue-url http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/test-queue \
  --wait-time-seconds 10
```
```
awslocal sqs delete-message \
  --queue-url http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/test-queue \
  --receipt-handle "<HERE>"
```
```
awslocal sqs get-queue-attributes \
  --queue-url http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/test-queue \
  --attribute-names QueueArn
```
## SNS
```
awslocal sns create-topic --name jobs-topic
```
```
awslocal sqs get-queue-attributes \
  --queue-url http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/test-queue \
  --attribute-names QueueArn
```
```
awslocal sns subscribe \
  --topic-arn arn:aws:sns:us-east-1:000000000000:jobs-topic \
  --protocol sqs \
  --notification-endpoint arn:aws:sqs:us-east-1:000000000000:test-queue
```
```
awslocal sns publish \
  --topic-arn arn:aws:sns:us-east-1:000000000000:jobs-topic \
  --message '{"type":"int","intVal":5}'
```

## Dynamo
### Commands
```
awslocal dynamodb create-table \
  --table-name job_results \
  --key-schema AttributeName=job_id,KeyType=HASH \
  --attribute-definitions AttributeName=job_id,AttributeType=S \
  --billing-mode PAY_PER_REQUEST \
  --region us-east-1
```