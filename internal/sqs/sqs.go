package sqs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Zupecki/go-patterns-aws/internal/jobs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
)

// SQS
type JobQueueMessage struct {
	Type   jobs.JobType `json:"type"`
	IDRaw  string       `json:"id"`
	IntVal int          `json:"intVal,omitempty"`
	StrVal string       `json:"strVal,omitempty"`
}

func PollSQS(ctx context.Context, queueURL string, jobChan chan<- jobs.Job) error {
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

				var jqm JobQueueMessage
				if err := json.Unmarshal([]byte(*m.Body), &jqm); err != nil {
					return err
				}

				job, err := parseJobQueueMessage(jqm)
				if err != nil {
					return err
				}

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

// helpers
func parseJobQueueMessage(jqm JobQueueMessage) (jobs.Job, error) {
	// prepare job id; check if empty, else ensure uuid format
	var jobID uuid.UUID
	if jqm.IDRaw == "" {
		jobID = uuid.New()
	} else {
		parsedID, err := uuid.Parse(jqm.IDRaw)
		if err != nil {
			jobID = uuid.New()
			fmt.Printf("queue item id (%s) has invalid format... overwriting with valid uuid: %+v\n", jqm.IDRaw, jobID)
			// log error, ignore or overwrite id (overwriting for demo)
		} else {
			jobID = parsedID
		}
	}

	var job jobs.Job
	switch jqm.Type {
	case jobs.JobTypeInt:
		job = jobs.JobProcessInt{
			ID:     jobID,
			IntVal: jqm.IntVal,
		}
	case jobs.JobTypeString:
		job = jobs.JobProcessString{
			ID:     jobID,
			StrVal: jqm.StrVal,
		}
	default:
		return nil, fmt.Errorf("unsupported queue message type for job type")
	}

	return job, nil
}
