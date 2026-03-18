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
	StrVal string
}

// results
type Result interface {
	ResultType() string
}

type ResultJobInt struct {
	ID     uuid.UUID
	IntVal int
}

func (r ResultJobInt) ResultType() string { return "int" }

type ResultJobStr struct {
	ID     uuid.UUID
	StrVal string
}

func (r ResultJobStr) ResultType() string { return "string" }

// store
type ResultStore interface {
	StoreResult(ctx context.Context, result Result) error
}

type PrintStore struct{}

func (s PrintStore) StoreResult(ctx context.Context, r Result) error {
	switch v := r.(type) {
	case ResultJobInt:
		fmt.Printf("INT RESULT: id=%s value=%d\n", v.ID, v.IntVal)
	case ResultJobStr:
		fmt.Printf("STRING RESULT: id=%s value=%q\n", v.ID, v.StrVal)
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
