package worker

import (
	"context"
	"fmt"

	"github.com/Zupecki/go-patterns-aws/internal/jobs"
)

func Worker(ctx context.Context, jobChan <-chan jobs.SQSJob, resultsChan chan<- jobs.SQSResult, ID int) error {
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Worker %d exiting due to early cancel\n", ID)
			return ctx.Err()
		case sqsJob, ok := <-jobChan:
			if !ok {
				fmt.Printf("Worker %d exiting: no more jobs\n", ID)
				return nil
			}

			fmt.Printf("Worker %d picked up job: %+v\n", ID, sqsJob.Job)

			result, err := sqsJob.Job.Process(ctx)
			if err != nil {
				return fmt.Errorf("job process error: %w", err)
			}

			sqsResult := jobs.SQSResult{
				Result:        result,
				QueueURL:      sqsJob.QueueURL,
				ReceiptHandle: sqsJob.ReceiptHandle,
			}

			select {
			case resultsChan <- sqsResult:
			case <-ctx.Done():
				fmt.Printf("Worker %d exiting due to early cancel\n", ID)
				return ctx.Err()
			}

			fmt.Printf("Worker %d finished job\n", ID)
		}
	}

}
