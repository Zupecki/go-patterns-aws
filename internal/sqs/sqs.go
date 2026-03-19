package sqs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Zupecki/go-patterns-aws/internal/jobs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
)

// SQS
type JobSQSMessage struct {
	Type   jobs.JobType `json:"type"`
	IDRaw  string       `json:"id"`
	IntVal int          `json:"intVal,omitempty"`
	StrVal string       `json:"strVal,omitempty"`
}

func NewLocalStackClient(ctx context.Context) (*awssqs.Client, error) {
	// create sqs client with AWS SDK
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		return nil, err
	}

	sqsClient := awssqs.NewFromConfig(cfg, func(o *awssqs.Options) {
		o.BaseEndpoint = aws.String("http://localhost:4566")
	})

	return sqsClient, nil
}

func SQSPoll(ctx context.Context, sqsClient *awssqs.Client, queueURL string, jobChan chan<- jobs.SQSJob) error {
	// long poll
	sqsParams := awssqs.ReceiveMessageInput{
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
			// poll
			out, err := sqsClient.ReceiveMessage(ctx, &sqsParams)
			if err != nil {
				return err
			}

			if len(out.Messages) == 0 {
				continue
			}

			// iterate over messages
			for _, m := range out.Messages {
				if m.Body == nil {
					continue
				}

				var jqm JobSQSMessage
				if err := json.Unmarshal([]byte(*m.Body), &jqm); err != nil {
					return err
				}

				if m.ReceiptHandle == nil {
					return fmt.Errorf("missing receipt handle")
				}

				sqsJob, err := parseJobQueueMessage(jqm, queueURL, *m.ReceiptHandle, *m.MessageId)
				if err != nil {
					return err
				}

				// cancel aware job loading
				select {
				case jobChan <- sqsJob:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}

func SQSDeleteMessage(ctx context.Context, sqsClient *awssqs.Client, queueURL string, receiptHandle string, messageID string) error {
	fmt.Println("Deleting message from queue with ID: ", messageID)

	sqsParams := awssqs.DeleteMessageInput{
		QueueUrl:      &queueURL,
		ReceiptHandle: &receiptHandle,
	}

	_, err := sqsClient.DeleteMessage(ctx, &sqsParams)
	if err != nil {
		return err
	}

	fmt.Println("Message deleted from queue")
	return nil
}

// helpers
func parseJobQueueMessage(jqm JobSQSMessage, queueURL string, receiptHandle string, messageID string) (jobs.SQSJob, error) {
	fmt.Println("Creating job from message id: ", messageID)
	var job jobs.Job
	switch jqm.Type {
	case jobs.JobTypeInt:
		job = jobs.JobProcessInt{
			ID:     uuid.New(),
			IntVal: jqm.IntVal,
		}
	case jobs.JobTypeString:
		job = jobs.JobProcessString{
			ID:     uuid.New(),
			StrVal: jqm.StrVal,
		}
	default:
		return jobs.SQSJob{}, fmt.Errorf("unsupported queue message type for job type")
	}

	sqsJob := jobs.SQSJob{
		MessageID:     messageID,
		Job:           job,
		QueueURL:      queueURL,
		ReceiptHandle: receiptHandle,
	}

	return sqsJob, nil
}
