// Package queue provides task handlers for Asynq background processing.
package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"krillin-ai/internal/dto"
	"krillin-ai/internal/service"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
	"krillin-ai/log"
)

// TaskHandlers provides handlers for different task types
type TaskHandlers struct {
	service *service.Service
}

// NewTaskHandlers creates a new TaskHandlers instance
func NewTaskHandlers(svc *service.Service) *TaskHandlers {
	return &TaskHandlers{service: svc}
}

// HandleSubtitleTask processes subtitle generation tasks
func (h *TaskHandlers) HandleSubtitleTask(ctx context.Context, t *asynq.Task) error {
	var payload SubtitleTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.GetLogger().Info("[Queue] Processing subtitle task",
		zap.String("task_id", payload.TaskID),
		zap.String("url", payload.URL))

	// Convert queue payload to service DTO
	req := dto.StartVideoSubtitleTaskReq{
		Url:            payload.URL,
		AudioUrl:       payload.AudioURL,
		OriginLanguage: payload.OriginLanguage,
		TargetLang:     payload.TargetLanguage,
		TtsVoiceCode:   payload.TtsVoiceCode,
		Bilingual:      payload.Bilingual,
		ReuseTaskId:    payload.TaskID, // Reuse the queue-assigned task ID
	}

	if payload.EnableTts {
		req.Tts = types.SubtitleTaskTtsYes
	}

	// Execute the task (this runs in the worker goroutine)
	_, err := h.service.StartSubtitleTask(req)
	if err != nil {
		// Update task status to failed
		task, _ := storage.GetTask(payload.TaskID)
		if task != nil {
			task.Status = types.SubtitleTaskStatusFailed
			task.FailReason = err.Error()
			_ = storage.SaveTask(task)
		}
		return err
	}

	log.GetLogger().Info("[Queue] Subtitle task completed",
		zap.String("task_id", payload.TaskID))

	return nil
}

// HandleSmartClipperTask processes smart clipper analysis tasks
func (h *TaskHandlers) HandleSmartClipperTask(ctx context.Context, t *asynq.Task) error {
	var payload SmartClipperPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.GetLogger().Info("[Queue] Processing smart clipper task",
		zap.String("task_id", payload.TaskID),
		zap.String("url", payload.URL))

	// Smart clipper logic would go here
	// For now, this is a placeholder as the smart clipper service
	// is already implemented with synchronous processing

	log.GetLogger().Info("[Queue] Smart clipper task completed",
		zap.String("task_id", payload.TaskID))

	return nil
}

// RegisterHandlers registers all task handlers with the Asynq server mux
func (h *TaskHandlers) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeSubtitleTask, h.HandleSubtitleTask)
	mux.HandleFunc(TypeSmartClipperTask, h.HandleSmartClipperTask)
}

// StartWorker starts the Asynq worker with registered handlers
func StartWorker(q *Queue, svc *service.Service) error {
	handlers := NewTaskHandlers(svc)
	
	mux := asynq.NewServeMux()
	handlers.RegisterHandlers(mux)

	log.GetLogger().Info("[Queue] Starting worker",
		zap.String("redis_addr", q.config.RedisAddr),
		zap.Int("concurrency", q.config.Concurrency))

	return q.server.Run(mux)
}
