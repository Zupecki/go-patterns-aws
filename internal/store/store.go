package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Zupecki/go-patterns-aws/internal/jobs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// NewLocalStackClient hard codes localstack aws config and returns dynamo client; update to external config load later
func NewLocalStackClient(ctx context.Context) (*dynamodb.Client, error) {
	// create dynamo client with AWS SDK
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		return nil, err
	}

	dynamoClient := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String("http://localhost:4566")
	})

	return dynamoClient, nil
}

type ResultStore interface {
	StoreResult(ctx context.Context, result jobs.Result, messageID string) error
}

type PrintStore struct{}

func (s PrintStore) StoreResult(ctx context.Context, r jobs.Result, messageID string) error {
	fmt.Printf("Job Result: %v for Message ID: %s\n", r, messageID)
	return nil
}

type DynamoStore struct {
	db        *dynamodb.Client
	tableName string
}

func NewDynamoStore(db *dynamodb.Client, tableName string) *DynamoStore {
	return &DynamoStore{
		db:        db,
		tableName: tableName,
	}
}

type ResultItem struct {
	JobID      string `dynamodbav:"job_id"`
	MessageID  string `dynamodbav:"message_id"`
	JobType    string `dynamodbav:"job_type"`
	ResultJSON string `dynamodbav:"result_json"`
	CreatedAt  string `dynamodbav:"created_at"`
}

func (s *DynamoStore) StoreResult(ctx context.Context, r jobs.Result, messageID string) error {
	resultJSON, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal result json: %w", err)
	}

	item := ResultItem{
		JobID:      r.JobID().String(),
		MessageID:  messageID,
		JobType:    string(r.JobType()),
		ResultJSON: string(resultJSON),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}

	putItemInput := dynamodb.PutItemInput{
		Item:      av,
		TableName: &s.tableName,
	}

	fmt.Printf("Storing job result in dynamo db for job id=%s, message id=%s", r.JobID().String(), messageID)
	_, err = s.db.PutItem(ctx, &putItemInput)
	if err != nil {
		return err
	}

	return nil
}
