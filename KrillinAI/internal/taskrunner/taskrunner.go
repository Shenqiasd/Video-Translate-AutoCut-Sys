package taskrunner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	"krillin-ai/internal/dto"
	"krillin-ai/internal/service"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
	"krillin-ai/log"
)

const (
	defaultQueueSize   = 128
	defaultConcurrency = 2
)

var (
	ErrRunnerStopped = errors.New("task runner stopped")
	ErrQueueFull     = errors.New("task queue is full")
)

// Config controls in-process task runner behavior.
type Config struct {
	QueueSize   int
	Concurrency int
}

// DefaultConfig returns a desktop-friendly default config.
func DefaultConfig() Config {
	return Config{
		QueueSize:   defaultQueueSize,
		Concurrency: defaultConcurrency,
	}
}

// SubtitleTaskPayload contains subtitle task enqueue data.
type SubtitleTaskPayload struct {
	TaskID         string `json:"task_id"`
	URL            string `json:"url"`
	AudioURL       string `json:"audio_url,omitempty"`
	OriginLanguage string `json:"origin_language"`
	TargetLanguage string `json:"target_language"`
	TtsVoiceCode   string `json:"tts_voice_code,omitempty"`
	EnableTts      bool   `json:"enable_tts"`
	Bilingual      int8   `json:"bilingual"`
	EmbedType      string `json:"embed_type,omitempty"`
}

// SmartClipperTaskPayload contains smart clipper task enqueue data.
type SmartClipperTaskPayload struct {
	TaskID string `json:"task_id"`
	URL    string `json:"url"`
}

// SmartClipperPayload keeps compatibility with the previous queue payload naming.
type SmartClipperPayload = SmartClipperTaskPayload

type queuedTaskType uint8

const (
	queuedTaskSubtitle queuedTaskType = iota + 1
	queuedTaskSmartClipper
)

type queuedTask struct {
	taskType     queuedTaskType
	subtitle     SubtitleTaskPayload
	smartClipper SmartClipperTaskPayload
}

// Runner executes queued tasks with in-memory workers.
type Runner struct {
	service *service.Service
	config  Config

	queue  chan queuedTask
	ctx    context.Context
	cancel context.CancelFunc

	workerWg sync.WaitGroup
	closed   atomic.Bool
}

// New creates and starts a task runner.
func New(svc *service.Service, cfg Config) *Runner {
	if svc == nil {
		svc = service.NewService()
	}

	cfg = normalizeConfig(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	runner := &Runner{
		service: svc,
		config:  cfg,
		queue:   make(chan queuedTask, cfg.QueueSize),
		ctx:     ctx,
		cancel:  cancel,
	}

	for i := 0; i < cfg.Concurrency; i++ {
		runner.workerWg.Add(1)
		go runner.worker(i + 1)
	}

	return runner
}

func normalizeConfig(cfg Config) Config {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = defaultQueueSize
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = defaultConcurrency
	}
	return cfg
}

// SubmitSubtitleTask queues a subtitle generation job.
func (r *Runner) SubmitSubtitleTask(payload SubtitleTaskPayload) error {
	if payload.URL == "" {
		return errors.New("subtitle task url is required")
	}

	return r.submit(queuedTask{
		taskType: queuedTaskSubtitle,
		subtitle: payload,
	}, payload.TaskID, "subtitle")
}

// SubmitSmartClipperTask queues a smart clipper analysis job.
func (r *Runner) SubmitSmartClipperTask(payload SmartClipperTaskPayload) error {
	if payload.URL == "" {
		return errors.New("smart clipper task url is required")
	}

	return r.submit(queuedTask{
		taskType:     queuedTaskSmartClipper,
		smartClipper: payload,
	}, payload.TaskID, "smart_clipper")
}

func (r *Runner) submit(task queuedTask, taskID, taskType string) error {
	if r.closed.Load() {
		return ErrRunnerStopped
	}

	select {
	case <-r.ctx.Done():
		return ErrRunnerStopped
	case r.queue <- task:
		log.GetLogger().Info("[TaskRunner] task submitted",
			zap.String("task_id", taskID),
			zap.String("task_type", taskType))
		return nil
	default:
		return ErrQueueFull
	}
}

func (r *Runner) worker(workerID int) {
	defer r.workerWg.Done()

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		select {
		case <-r.ctx.Done():
			return
		case task := <-r.queue:
			r.processTask(workerID, task)
		}
	}
}

func (r *Runner) processTask(workerID int, task queuedTask) {
	var err error
	var taskID string
	var taskType string

	switch task.taskType {
	case queuedTaskSubtitle:
		taskID = task.subtitle.TaskID
		taskType = "subtitle"
		err = r.processSubtitleTask(task.subtitle)
	case queuedTaskSmartClipper:
		taskID = task.smartClipper.TaskID
		taskType = "smart_clipper"
		err = r.processSmartClipperTask(task.smartClipper)
	default:
		err = fmt.Errorf("unsupported task type: %d", task.taskType)
	}

	if err != nil {
		log.GetLogger().Error("[TaskRunner] task failed",
			zap.Int("worker_id", workerID),
			zap.String("task_id", taskID),
			zap.String("task_type", taskType),
			zap.Error(err))
		return
	}

	log.GetLogger().Info("[TaskRunner] task completed",
		zap.Int("worker_id", workerID),
		zap.String("task_id", taskID),
		zap.String("task_type", taskType))
}

func (r *Runner) processSubtitleTask(payload SubtitleTaskPayload) error {
	if r.service == nil {
		return errors.New("service not initialized")
	}

	req := dto.StartVideoSubtitleTaskReq{
		Url:                    payload.URL,
		AudioUrl:               payload.AudioURL,
		OriginLanguage:         payload.OriginLanguage,
		TargetLang:             payload.TargetLanguage,
		TtsVoiceCode:           payload.TtsVoiceCode,
		Bilingual:              uint8(payload.Bilingual),
		EmbedSubtitleVideoType: payload.EmbedType,
		ReuseTaskId:            payload.TaskID,
	}

	if payload.EnableTts {
		req.Tts = types.SubtitleTaskTtsYes
	}

	_, err := r.service.StartSubtitleTask(req)
	if err != nil {
		r.markSubtitleTaskFailed(payload.TaskID, err)
		return err
	}

	return nil
}

func (r *Runner) processSmartClipperTask(payload SmartClipperTaskPayload) error {
	if r.service == nil {
		return errors.New("service not initialized")
	}

	_, err := r.service.AnalyzeVideo(dto.SmartClipperAnalyzeReq{Url: payload.URL})
	return err
}

func (r *Runner) markSubtitleTaskFailed(taskID string, taskErr error) {
	if taskID == "" {
		return
	}

	task, err := storage.GetTask(taskID)
	if err != nil || task == nil {
		return
	}

	task.Status = types.SubtitleTaskStatusFailed
	task.FailReason = taskErr.Error()
	task.StatusMsg = "任务失败 Failed"
	_ = storage.SaveTask(task)
}

// Close stops workers and rejects new tasks.
func (r *Runner) Close() {
	if !r.closed.CompareAndSwap(false, true) {
		return
	}

	r.cancel()
	r.workerWg.Wait()
}

// Pending returns the number of queued tasks waiting for workers.
func (r *Runner) Pending() int {
	return len(r.queue)
}
