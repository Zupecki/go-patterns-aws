package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type JobType string

const (
	JobTypeInt    JobType = "int"
	JobTypeString JobType = "string"
)

type Result interface {
	ResultJobType() JobType
}

type ResultJobInt struct {
	ID     uuid.UUID `json:"id"`
	IntVal int       `json:"intVal"`
}

func (r ResultJobInt) ResultJobType() JobType { return JobTypeInt }
func (r ResultJobInt) String() string {
	return fmt.Sprintf("id=%s resulttype=%s value=%d", r.ID.String(), r.ResultJobType(), r.IntVal)
}

type ResultJobString struct {
	ID     uuid.UUID `json:"id"`
	StrVal string    `json:"strVal"`
}

func (r ResultJobString) ResultJobType() JobType { return JobTypeString }
func (r ResultJobString) String() string {
	return fmt.Sprintf("id=%s resulttype=%s value=%s", r.ID.String(), r.ResultJobType(), r.StrVal)
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
