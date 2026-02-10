// Package queue provides background task processing using Asynq.
// It supports reliable task queueing with retry logic and persistence.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"krillin-ai/config"
	"krillin-ai/log"
)

// Task type names
const (
	TypeSubtitleTask    = "subtitle:process"
	TypeSmartClipperTask = "clipper:analyze"
)

// SubtitleTaskPayload contains the data for subtitle processing task
type SubtitleTaskPayload struct {
	TaskID           string `json:"task_id"`
	URL              string `json:"url"`
	AudioURL         string `json:"audio_url,omitempty"`
	OriginLanguage   string `json:"origin_language"`
	TargetLanguage   string `json:"target_language"`
	TtsVoiceCode     string `json:"tts_voice_code,omitempty"`
	EnableTts        bool   `json:"enable_tts"`
	Bilingual        int8   `json:"bilingual"`
	EmbedType        string `json:"embed_type,omitempty"`
}

// SmartClipperPayload contains the data for smart clipper task
type SmartClipperPayload struct {
	TaskID string `json:"task_id"`
	URL    string `json:"url"`
}

// QueueConfig holds Redis configuration for Asynq
type QueueConfig struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	Concurrency   int
}

// Queue manages task enqueueing and processing
type Queue struct {
	client *asynq.Client
	server *asynq.Server
	config QueueConfig
}

// DefaultConfig returns default queue configuration
// Uses in-memory mode if Redis is not configured
func DefaultConfig() QueueConfig {
	return QueueConfig{
		RedisAddr:   "localhost:6379",
		RedisDB:     0,
		Concurrency: 3,
	}
}

// NewQueue creates a new Queue instance
func NewQueue(cfg QueueConfig) *Queue {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	client := asynq.NewClient(redisOpt)

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: cfg.Concurrency,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				// Exponential backoff: 10s, 20s, 40s, 80s, ...
				return time.Duration(10<<uint(n)) * time.Second
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.GetLogger().Error("Task failed",
					zap.String("type", task.Type()),
					zap.ByteString("payload", task.Payload()),
					zap.Error(err))
			}),
		},
	)

	return &Queue{
		client: client,
		server: server,
		config: cfg,
	}
}

// EnqueueSubtitleTask adds a subtitle processing task to the queue
func (q *Queue) EnqueueSubtitleTask(payload SubtitleTaskPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TypeSubtitleTask, data,
		asynq.MaxRetry(3),
		asynq.Timeout(30*time.Minute),
		asynq.Queue("default"),
	)

	info, err := q.client.Enqueue(task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.GetLogger().Info("Task enqueued",
		zap.String("task_id", payload.TaskID),
		zap.String("queue_id", info.ID),
		zap.String("queue", info.Queue))

	return nil
}

// EnqueueSmartClipperTask adds a smart clipper task to the queue
func (q *Queue) EnqueueSmartClipperTask(payload SmartClipperPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TypeSmartClipperTask, data,
		asynq.MaxRetry(2),
		asynq.Timeout(10*time.Minute),
		asynq.Queue("default"),
	)

	info, err := q.client.Enqueue(task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.GetLogger().Info("Smart Clipper task enqueued",
		zap.String("task_id", payload.TaskID),
		zap.String("queue_id", info.ID))

	return nil
}

// Close gracefully shuts down the queue
func (q *Queue) Close() error {
	if err := q.client.Close(); err != nil {
		return err
	}
	q.server.Shutdown()
	return nil
}

// Client returns the underlying Asynq client for advanced usage
func (q *Queue) Client() *asynq.Client {
	return q.client
}

// Server returns the underlying Asynq server for advanced usage
func (q *Queue) Server() *asynq.Server {
	return q.server
}
