package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// jobs
type Job interface {
	Process(ctx context.Context) (Result, error)
}

type JobProcessString struct {
	ID     uuid.UUID
	StrVal string
}

type JobProcessInt struct {
	ID     uuid.UUID
	IntVal int
}

// results
type ResultType string

const (
	ResultTypeInt ResultType = "int"
	ResultTypeStr ResultType = "string"
)

type Result interface {
	ResultType() ResultType
}

type ResultJobInt struct {
	ID     uuid.UUID
	IntVal int
}

func (r ResultJobInt) ResultType() ResultType { return ResultTypeInt }
func (r ResultJobInt) String() string {
	return fmt.Sprintf("id=%s resulttype=%s value=%d", r.ID.String(), r.ResultType(), r.IntVal)
}

type ResultJobStr struct {
	ID     uuid.UUID
	StrVal string
}

func (r ResultJobStr) ResultType() ResultType { return ResultTypeStr }
func (r ResultJobStr) String() string {
	return fmt.Sprintf("id=%s resulttype=%s value=%s", &r.ID.String(), r.ResultType(), r.StrVal)
}

// store
type ResultStore interface {
	StoreResult(ctx context.Context, result Result) error
}

type PrintStore struct{}

func (s PrintStore) StoreResult(ctx context.Context, r Result) error {
	switch r.ResultType() {
	case ResultTypeInt, ResultTypeStr:
		fmt.Printf("Job Result: %s\n", r)
	default:
		return fmt.Errorf("unknown result type")
	}

	return nil
}

// add NoSQL store later

func run(ctx context.Context) error {
	jobChan := make(chan<- Job)
	resultsChan := make(<-chan Result)

	return nil
}
