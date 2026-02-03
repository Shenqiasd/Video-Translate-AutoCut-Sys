package storage

import (
	"errors"
	"krillin-ai/internal/types"

	"gorm.io/gorm"
)

// Deprecated: Migrating to DB
// var SubtitleTasks = sync.Map{} 

func SaveTask(task *types.SubtitleTask) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	// Upsert: Create or Update based on primary key (Id) or TaskId logic
	// Since TaskId is unique but Id is primary key, we search by TaskId first
	var existing types.SubtitleTask
	result := DB.Where("task_id = ?", task.TaskId).First(&existing)
	
	if result.Error == nil {
		// Update
		task.Id = existing.Id // Preserve ID
		return DB.Save(task).Error
	} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Create
		return DB.Create(task).Error
	}
	return result.Error
}

func GetTask(taskId string) (*types.SubtitleTask, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var task types.SubtitleTask
	if err := DB.Preload("SubtitleInfos").Where("task_id = ?", taskId).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func GetTaskHistory(limit int) ([]types.SubtitleTask, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var tasks []types.SubtitleTask
	if err := DB.Preload("SubtitleInfos").Order("create_time desc").Limit(limit).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func DeleteTask(taskId string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Where("task_id = ?", taskId).Delete(&types.SubtitleTask{}).Error
}

// MarkStaleTasks marks all "running" tasks (status=1) as "failed" (status=3)
// This should be called on server startup to clean up zombie tasks
func MarkStaleTasks() (int64, error) {
	if DB == nil {
		return 0, errors.New("database not initialized")
	}
	// Status 1 = running, Status 3 = failed
	result := DB.Model(&types.SubtitleTask{}).
		Where("status = ?", 1).
		Updates(map[string]interface{}{
			"status":      3,
			"fail_reason": "服务重启，任务被中断 Task interrupted by server restart",
			"status_msg":  "任务超时/中断 Task Timeout/Interrupted",
		})
	return result.RowsAffected, result.Error
}
