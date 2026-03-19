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

// Job Result
type Result interface {
	JobType() JobType
}

type ResultJobInt struct {
	ID     uuid.UUID `json:"id"`
	IntVal int       `json:"intVal"`
}

func (r ResultJobInt) JobType() JobType { return JobTypeInt }
func (r ResultJobInt) String() string {
	return fmt.Sprintf("id=%s resulttype=%s value=%d", r.ID.String(), r.JobType(), r.IntVal)
}

type ResultJobString struct {
	ID     uuid.UUID `json:"id"`
	StrVal string    `json:"strVal"`
}

func (r ResultJobString) JobType() JobType { return JobTypeString }
func (r ResultJobString) String() string {
	return fmt.Sprintf("id=%s resulttype=%s value=%s", r.ID.String(), r.JobType(), r.StrVal)
}

// Job
type Job interface {
	JobType() JobType
	Process(ctx context.Context) (Result, error)
}

type JobProcessInt struct {
	ID     uuid.UUID
	IntVal int
}

func (j JobProcessInt) JobType() JobType { return JobTypeInt }

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

type JobProcessString struct {
	ID     uuid.UUID
	StrVal string
}

func (j JobProcessString) JobType() JobType { return JobTypeString }

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
