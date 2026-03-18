package main

import (
	"context"
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

type JobType string

const (
	JobTypeInt JobType = "int"
	JobTypeStr JobType = "string"
)

type Job struct {
	ID     uuid.UUID
	Type   JobType
	IntVal int
	StrVal string
}

type Result struct {
	Type   string // "int" or "string"
	IntVal int
	StrVal string
}

func run(ctx context.Context) error {
	jobChan := make(chan<- Job)
	resultsChan := make(<-chan Result)

	return nil
}
