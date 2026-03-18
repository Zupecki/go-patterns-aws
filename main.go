package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Zupecki/go-patterns-aws/internal/jobs"
	"github.com/Zupecki/go-patterns-aws/internal/worker"
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

// store
type ResultStore interface {
	StoreResult(ctx context.Context, result jobs.Result) error
}

type PrintStore struct{}

func (s PrintStore) StoreResult(ctx context.Context, r jobs.Result) error {
	fmt.Printf("Job Result: %v\n", r)
	return nil
}

// add NoSQL store later
func run(ctx context.Context) error {
	jobChan := make(chan jobs.Job)
	resultsChan := make(chan jobs.Result)
	errChan := make(chan error, 1) // buffer with 1, so worker
	numWorkers := 5
	numJobs := 10

	errorGroup, ctx := errgroup.WithContext(ctx)

	// spawn workers with early cancel via errorgroup and context
	for i := 1; i <= numWorkers; i++ {
		i := i
		errorGroup.Go(func() error {
			return worker.Worker(ctx, jobChan, resultsChan, i)
		})
	}

	// coordination routine, ensure results channel closed when jobs done (error group wait group)
	go func() {
		defer close(resultsChan)
		defer close(errChan)

		err := errorGroup.Wait()
		errChan <- err
	}()

	// job producer
	go func() {
		defer close(jobChan)

		for i := 1; i <= numJobs; i++ {
			var job jobs.Job
			jobID := uuid.New()

			if i%2 == 0 {
				job = jobs.JobProcessInt{
					ID:     jobID,
					IntVal: i,
				}
			} else {
				job = jobs.JobProcessString{
					ID:     jobID,
					StrVal: fmt.Sprintf("i=%d", i),
				}
			}

			fmt.Printf("Job Created: %+v\n", job)

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
