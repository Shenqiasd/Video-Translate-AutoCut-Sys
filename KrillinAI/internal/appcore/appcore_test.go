package appcore

import (
	"context"
	"reflect"
	"testing"
)

var _ JobHandle = (*stubHandle)(nil)
var _ Runner = (*stubRunner)(nil)

type stubHandle struct {
	id       string
	events   chan JobEvent
	result   chan JobResult
	canceled bool
}

func (h *stubHandle) ID() string {
	return h.id
}

func (h *stubHandle) Events() <-chan JobEvent {
	return h.events
}

func (h *stubHandle) Result() <-chan JobResult {
	return h.result
}

func (h *stubHandle) Cancel() error {
	h.canceled = true
	return nil
}

type stubRunner struct {
	handle  JobHandle
	lastCtx context.Context
	lastReq JobRequest
}

func (r *stubRunner) Submit(ctx context.Context, req JobRequest) (JobHandle, error) {
	r.lastCtx = ctx
	r.lastReq = req
	return r.handle, nil
}

func TestJobStageStringAndTerminal(t *testing.T) {
	testCases := []struct {
		stage      JobStage
		wantString string
		terminal   bool
	}{
		{stage: JobStageQueued, wantString: "queued", terminal: false},
		{stage: JobStagePreparing, wantString: "preparing", terminal: false},
		{stage: JobStageProcessing, wantString: "processing", terminal: false},
		{stage: JobStageFinalizing, wantString: "finalizing", terminal: false},
		{stage: JobStageSucceeded, wantString: "succeeded", terminal: true},
		{stage: JobStageFailed, wantString: "failed", terminal: true},
		{stage: JobStageCanceled, wantString: "canceled", terminal: true},
		{stage: JobStage(255), wantString: "unknown", terminal: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.wantString, func(t *testing.T) {
			if got := tc.stage.String(); got != tc.wantString {
				t.Fatalf("JobStage.String() = %q, want %q", got, tc.wantString)
			}
			if got := tc.stage.IsTerminal(); got != tc.terminal {
				t.Fatalf("JobStage.IsTerminal() = %t, want %t", got, tc.terminal)
			}
		})
	}
}

func TestRunnerAndHandleContracts(t *testing.T) {
	eventCh := make(chan JobEvent, 1)
	resultCh := make(chan JobResult, 1)

	handle := &stubHandle{
		id:     "job-42",
		events: eventCh,
		result: resultCh,
	}

	runner := &stubRunner{handle: handle}

	req := JobRequest{
		ID:         "job-42",
		InputPath:  "/tmp/input.mp4",
		WorkingDir: "/tmp/workdir",
		OutputDir:  "/tmp/output",
		Args: map[string]any{
			"language": "en",
		},
		Metadata: map[string]string{
			"source": "desktop",
		},
	}

	ctx := context.Background()
	jobHandle, err := runner.Submit(ctx, req)
	if err != nil {
		t.Fatalf("Submit() returned unexpected error: %v", err)
	}
	if jobHandle.ID() != req.ID {
		t.Fatalf("jobHandle.ID() = %q, want %q", jobHandle.ID(), req.ID)
	}
	if runner.lastCtx == nil {
		t.Fatal("runner did not receive context")
	}
	if !reflect.DeepEqual(runner.lastReq, req) {
		t.Fatalf("runner received request %+v, want %+v", runner.lastReq, req)
	}

	expectedEvent := JobEvent{
		JobID:   req.ID,
		Stage:   JobStageProcessing,
		Message: "working",
	}
	eventCh <- expectedEvent
	gotEvent := <-jobHandle.Events()
	if !reflect.DeepEqual(gotEvent, expectedEvent) {
		t.Fatalf("Events() yielded %+v, want %+v", gotEvent, expectedEvent)
	}

	expectedResult := JobResult{
		JobID:      req.ID,
		Stage:      JobStageSucceeded,
		OutputPath: "/tmp/output/final.mp4",
	}
	resultCh <- expectedResult
	gotResult := <-jobHandle.Result()
	if !reflect.DeepEqual(gotResult, expectedResult) {
		t.Fatalf("Result() yielded %+v, want %+v", gotResult, expectedResult)
	}

	if err := jobHandle.Cancel(); err != nil {
		t.Fatalf("Cancel() returned unexpected error: %v", err)
	}
	if !handle.canceled {
		t.Fatal("Cancel() did not update handle state")
	}
}
