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
	result := ResultJobString{
		ID:     j.ID,
		StrVal: fmt.Sprintf("String processed: %s", j.StrVal),
	}
	return result, nil
}

type JobProcessInt struct {
	ID     uuid.UUID
	IntVal int
}

func (j JobProcessInt) Process(ctx context.Context) (Result, error) {
	result := ResultJobInt{
		ID:     j.ID,
		IntVal: j.IntVal * 2,
	}
	return result, nil
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
func worker(ctx context.Context, jobChan <-chan Job, resultsChan chan<- Result, ID int) error {
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Worker %d exiting due to early cancel", ID)
			return ctx.Err()
		case job, ok := <-jobChan:
			if !ok {
				return nil // job channel closed; no more jobs
			}

			result, err := job.Process(ctx)
			if err != nil {
				return fmt.Errorf("job process error: %w", err)
			}

			select {
			case resultsChan <- result:
			case <-ctx.Done():
				fmt.Printf("Worker %d exiting due to early cancel", ID)
				return ctx.Err()
			}
		}
	}

}

func run(ctx context.Context) error {
	jobChan := make(chan Job)
	resultsChan := make(chan Result)
	errChan := make(chan error, 1) // buffer with 1, so worker
	numWorkers := 5
	numJobs := 10

	errorGroup, ctx := errgroup.WithContext(ctx)

	// spawn workers with early cancel via errorgroup and context
	for i := 1; i <= numWorkers; i++ {
		i := i
		errorGroup.Go(func() error {
			return worker(ctx, jobChan, resultsChan, i)
		})
	}

	// coordination routine, ensure results channel closed when jobs done (error group wait group)
	go func() {
		defer close(resultsChan)

		err := errorGroup.Wait()
		errChan <- err
	}()

	// job producer
	go func() {
		defer close(jobChan)

		for i := 1; i <= numJobs; i++ {
			var job Job

			if i%2 == 0 {
				job = JobProcessInt{
					ID:     uuid.New(),
					IntVal: i,
				}
			} else {
				job = JobProcessString{
					ID:     uuid.New(),
					StrVal: fmt.Sprintf("i=%d", i),
				}
			}

			// cancel aware job loading
			select {
			case jobChan <- job:
			case <-ctx.Done():
				return
			}
		}
	}()

	// results consumer
	store := PrintStore{}
	for result := range resultsChan {
		err := store.StoreResult(ctx, result)
		if err != nil {
			return err
		}
	}

	// errChan will only ever be nil or the sync.Once error from errorgroup
	return <-errChan
}
