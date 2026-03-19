package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Zupecki/go-patterns-aws/internal/jobs"
	"github.com/Zupecki/go-patterns-aws/internal/store"
	"github.com/Zupecki/go-patterns-aws/internal/worker"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
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

func run(ctx context.Context) error {
	jobChan := make(chan jobs.Job)
	resultsChan := make(chan jobs.Result)
	errChan := make(chan error, 1) // buffer 1 so cleanup goroutine can send final error without blocking
	numWorkers := 5
	//numJobs := 10

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
	go pollSQS(
		ctx,
		"http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/test-queue",
		jobChan,
	)

	// results consumer
	// may want to cancel context if consumer issues, drain workers etc
	resultStore := store.PrintStore{}
	err := resultsConsumer(ctx, resultStore, resultsChan)
	if err != nil {
		return err
	}

	// errChan will only ever be nil or the sync.Once error from errorgroup
	return <-errChan
}

func resultsCleanup(errorGroup *errgroup.Group, resultsChan chan<- jobs.Result, errChan chan<- error) {
	defer close(resultsChan)
	defer close(errChan)

	err := errorGroup.Wait()
	errChan <- err
}

func resultsConsumer(ctx context.Context, store store.ResultStore, resultsChan <-chan jobs.Result) error {
	for result := range resultsChan {
		err := store.StoreResult(ctx, result)
		if err != nil {
			return err
		}
	}

	return nil
}

func produceTestJobs(ctx context.Context, jobChan chan<- jobs.Job, numJobs int) {
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
}

// SQS
type QueueMessage struct {
	Type   string `json:"type"`
	IntVal int    `json:"intVal,omitempty"`
	StrVal string `json:"strVal,omitempty"`
}

func pollSQS(ctx context.Context, queueURL string, jobChan chan<- jobs.Job) error {
	// create sqs client with AWS SDK
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		return err
	}

	sqsClient := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String("http://localhost:4566")
	})

	// long poll
	sqsParams := sqs.ReceiveMessageInput{
		QueueUrl:            &queueURL,
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     15,
	}

	// cancel aware polling
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			out, err := sqsClient.ReceiveMessage(ctx, &sqsParams)
			if err != nil {
				return err
			}

			if len(out.Messages) == 0 {
				continue
			}

			for _, m := range out.Messages {
				if m.Body == nil {
					continue
				}

				// parse into Queue struct
				var qm QueueMessage
				if err := json.Unmarshal([]byte(*m.Body), &qm); err != nil {
					return err
				}

				// parse into Job
				var job jobs.Job
				switch qm.Type {
				case "int":
					job = jobs.JobProcessInt{
						ID:     uuid.New(),
						IntVal: qm.IntVal,
					}
				case "string":
					job = jobs.JobProcessString{
						ID:     uuid.New(),
						StrVal: qm.StrVal,
					}
				default:
					return fmt.Errorf("unsupported queue message type for job type")
				}

				// load Job onto jobChan
				fmt.Println("Job: ", job)

				// cancel aware job loading
				select {
				case jobChan <- job:
				case <-ctx.Done():
					return ctx.Err()
				}

				// add delete sqs job message once load successful
			}
		}
	}
}
