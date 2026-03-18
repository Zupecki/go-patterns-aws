package worker

import (
	"context"
	"fmt"

	"github.com/Zupecki/go-patterns-aws/internal/jobs"
)

// func NewWorker() func(ctx context.Context, jobChan <-chan jobs.Job, resultsChan chan<- jobs.Result, ID int) error {
// 	return worker
// }

func Worker(ctx context.Context, jobChan <-chan jobs.Job, resultsChan chan<- jobs.Result, ID int) error {
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Worker %d exiting due to early cancel\n", ID)
			return ctx.Err()
		case job, ok := <-jobChan:
			if !ok {
				fmt.Printf("Worker %d exiting: no more jobs\n", ID)
				return nil
			}

			fmt.Printf("Worker %d picked up job\n", ID)

			result, err := job.Process(ctx)
			if err != nil {
				return fmt.Errorf("job process error: %w", err)
			}

			select {
			case resultsChan <- result:
			case <-ctx.Done():
				fmt.Printf("Worker %d exiting due to early cancel\n", ID)
				return ctx.Err()
			}

			fmt.Printf("Worker %d finished job\n", ID)
		}
	}

}
