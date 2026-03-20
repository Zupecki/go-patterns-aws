package store

import (
	"context"
	"fmt"

	"github.com/Zupecki/go-patterns-aws/internal/jobs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

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
	StoreResult(ctx context.Context, result jobs.Result) error
}

type PrintStore struct{}

func (s PrintStore) StoreResult(ctx context.Context, r jobs.Result) error {
	fmt.Printf("Job Result: %v\n", r)
	return nil
}

type DynamoStore struct {
	db *dynamodb.Client
}

func (s DynamoStore) StoreResults(ctx context.Context, r jobs.Result) error {
	fmt.Printf("Job Result: %v\n", r)
	return nil
}
