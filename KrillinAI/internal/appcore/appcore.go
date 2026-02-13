package appcore

import (
	"context"
	"time"
)

type JobRequest struct {
	ID         string
	InputPath  string
	WorkingDir string
	OutputDir  string
	Args       map[string]any
	Metadata   map[string]string
}

type JobStage uint8

const (
	JobStageQueued JobStage = iota + 1
	JobStagePreparing
	JobStageProcessing
	JobStageFinalizing
	JobStageSucceeded
	JobStageFailed
	JobStageCanceled
)

func (s JobStage) String() string {
	switch s {
	case JobStageQueued:
		return "queued"
	case JobStagePreparing:
		return "preparing"
	case JobStageProcessing:
		return "processing"
	case JobStageFinalizing:
		return "finalizing"
	case JobStageSucceeded:
		return "succeeded"
	case JobStageFailed:
		return "failed"
	case JobStageCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

func (s JobStage) IsTerminal() bool {
	return s == JobStageSucceeded || s == JobStageFailed || s == JobStageCanceled
}

type JobProgress struct {
	Stage     JobStage
	Current   int64
	Total     int64
	Percent   float64
	Message   string
	UpdatedAt time.Time
}

type JobEvent struct {
	JobID      string
	Stage      JobStage
	Progress   *JobProgress
	Message    string
	Err        error
	OccurredAt time.Time
}

type JobResult struct {
	JobID      string
	Stage      JobStage
	OutputPath string
	Artifacts  map[string]string
	StartedAt  time.Time
	FinishedAt time.Time
	Err        error
}

type JobHandle interface {
	ID() string
	Events() <-chan JobEvent
	Result() <-chan JobResult
	Cancel() error
}

type Runner interface {
	Submit(ctx context.Context, req JobRequest) (JobHandle, error)
}
