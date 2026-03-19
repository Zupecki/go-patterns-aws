package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Zupecki/go-patterns-aws/internal/jobs"
	"github.com/Zupecki/go-patterns-aws/internal/sqs"
	"github.com/Zupecki/go-patterns-aws/internal/store"
	"github.com/Zupecki/go-patterns-aws/internal/worker"
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

func run(ctx context.Context) error {
	jobChan := make(chan jobs.SQSJob)
	resultsChan := make(chan jobs.SQSResult)
	errChan := make(chan error, 1) // buffer 1 so cleanup goroutine can send final error without blocking
	numWorkers := 5

	sqsClient, err := sqs.NewLocalStackClient(ctx)
	if err != nil {
		return err
	}

	errorGroup, ctx := errgroup.WithContext(ctx)

	// spawn workers with early cancel via errorgroup and context
	for i := 1; i <= numWorkers; i++ {
		i := i
		errorGroup.Go(func() error {
			return worker.Worker(ctx, jobChan, resultsChan, i)
		})
	}

	// coordination routine, ensure results channel closed when jobs done (error group wait group)
	go resultsCleanup(errorGroup, resultsChan, errChan)

	// job producer
	//go produceTestJobs(ctx, jobChan, numJobs)
	go sqs.SQSPoll(
		ctx,
		sqsClient,
		"http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/test-queue",
		jobChan,
	)

	// results consumer
	resultStore := store.PrintStore{}
	err = resultsConsumer(ctx, resultStore, resultsChan)
	if err != nil {
		return err
	}

	// errChan will only ever be nil or the sync.Once error from errorgroup
	return <-errChan
}

func resultsCleanup(errorGroup *errgroup.Group, resultsChan chan<- jobs.SQSResult, errChan chan<- error) {
	defer close(resultsChan)
	defer close(errChan)

	err := errorGroup.Wait()
	errChan <- err
}

func resultsConsumer(ctx context.Context, store store.ResultStore, resultsChan <-chan jobs.SQSResult) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sqsResult, ok := <-resultsChan:
			if !ok {
				return nil
			}

			err := store.StoreResult(ctx, sqsResult.Result)
			if err != nil {
				return err
			}

			// delete message queue item on success
			err = sqs.SQSDeleteMessage(sqsResult.QueueURL, sqsResult.ReceiptHandle)
			if err != nil {
				return err
			}
		}
	}
}
