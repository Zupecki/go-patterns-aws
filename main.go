package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
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

func (j JobProcessString) Process(ctx context.Context) (Result, error) {
	fmt.Println("PROCESS STRING JOB")
	return nil, nil
}

type JobProcessInt struct {
	ID     uuid.UUID
	IntVal int
}

func (j JobProcessInt) Process(ctx context.Context) (Result, error) {
	fmt.Println("PROCESS INT JOB")
	return nil, nil
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

type ResultJobString struct {
	ID     uuid.UUID
	StrVal string
}

func (r ResultJobString) ResultType() ResultType { return ResultTypeStr }
func (r ResultJobString) String() string {
	return fmt.Sprintf("id=%s resulttype=%s value=%s", r.ID.String(), r.ResultType(), r.StrVal)
}

// store
type ResultStore interface {
	StoreResult(ctx context.Context, result Result) error
}

type PrintStore struct{}

func (s PrintStore) StoreResult(ctx context.Context, r Result) error {
	switch r.ResultType() {
	case ResultTypeInt, ResultTypeStr:
		fmt.Printf("Job Result: %v\n", r)
	default:
		return fmt.Errorf("unknown result type")
	}

	return nil
}

// add NoSQL store later

// worker
func worker(ctx context.Context, jobChan <-chan Job, resultsChan chan<- Result) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case job, ok := <-jobChan:
			if !ok {
				return fmt.Errorf("job chan closed; no more jobs")
			}

			result, err := job.Process(ctx)
			if err != nil {
				return fmt.Errorf("job process error: %w", err)
			}

			select {
			case resultsChan <- result:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

}

func run(ctx context.Context) error {
	jobChan := make(chan Job)
	resultsChan := make(chan Result)
	numWorkers := 5

	errorGroup, ctx := errgroup.WithContext(ctx)

	// spawn workers with early cancel via errorgroup and context
	for i := 1; i <= numWorkers; i++ {
		errorGroup.Go(func() error {
			return worker(ctx, jobChan, resultsChan)
		})
	}

	// coordination routine, ensure results channel closed when jobs done (error group wait group)
	go func() {
		defer close(resultsChan)
		err := errorGroup.Wait()
		fmt.Println(err)
	}()

	// job producer

	// results consumer

	return nil
}
