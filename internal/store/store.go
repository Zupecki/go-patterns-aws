package store

import (
	"context"
	"fmt"

	"github.com/Zupecki/go-patterns-aws/internal/jobs"
)

type ResultStore interface {
	StoreResult(ctx context.Context, result jobs.Result) error
}

type PrintStore struct{}

func (s PrintStore) StoreResult(ctx context.Context, r jobs.Result) error {
	fmt.Printf("Job Result: %v\n", r)
	return nil
}
