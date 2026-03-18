package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ResultType string

const (
	ResultTypeInt    ResultType = "int"
	ResultTypeString ResultType = "string"
)

type Result interface {
	ResultType() ResultType
}

type ResultJobInt struct {
	ID     uuid.UUID
	IntVal int
}

func (r ResultJobInt) ResultType() ResultType { return ResultTypeInt }
func (r ResultJobInt) String() string {
	return fmt.Sprintf("id=%s resulttype=%s value=%d", r.ID.String(), r.ResultType(), r.IntVal)
}

type ResultJobString struct {
	ID     uuid.UUID
	StrVal string
}

func (r ResultJobString) ResultType() ResultType { return ResultTypeString }
func (r ResultJobString) String() string {
	return fmt.Sprintf("id=%s resulttype=%s value=%s", r.ID.String(), r.ResultType(), r.StrVal)
}

type Job interface {
	Process(ctx context.Context) (Result, error)
}

type JobProcessString struct {
	ID     uuid.UUID
	StrVal string
}

func (j JobProcessString) Process(ctx context.Context) (Result, error) {
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	result := ResultJobString{
		ID:     j.ID,
		StrVal: fmt.Sprintf("String processed: %s", j.StrVal),
	}

	return result, nil
}

type JobProcessInt struct {
	ID     uuid.UUID
	IntVal int
}

func (j JobProcessInt) Process(ctx context.Context) (Result, error) {
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	result := ResultJobInt{
		ID:     j.ID,
		IntVal: j.IntVal * 2,
	}

	return result, nil
}
