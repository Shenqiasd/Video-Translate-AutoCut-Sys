package handler

import (
	"krillin-ai/internal/deps"
	"krillin-ai/internal/dto"
	"krillin-ai/internal/response"
	"krillin-ai/internal/service"
	"krillin-ai/internal/storage"
	"krillin-ai/log"
	apperrors "krillin-ai/pkg/errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h Handler) StartSubtitleTask(c *gin.Context) {
	var req dto.StartVideoSubtitleTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.GetLogger().Error("StartSubtitleTask ShouldBindJSON err", zap.Error(err))
		response.ErrorResponse(c, apperrors.Wrap(apperrors.CodeInvalidParams, "参数错误 Invalid parameters", err))
		return
	}
	log.GetLogger().Info("StartSubtitleTask received request", zap.Any("req", req))

	// 检查配置是否需要重新初始化
	if configUpdated {
		log.GetLogger().Info("检测到配置更新，重新初始化服务")
		deps.CheckDependency()
		h.Service = service.NewService()
		configUpdated = false
	}

	svc := h.Service

	data, err := svc.StartSubtitleTask(req)
	if err != nil {
		response.ErrorResponse(c, err)
		return
	}
	response.Success(c, data)
}

func (h Handler) GetSubtitleTask(c *gin.Context) {
	var req dto.GetVideoSubtitleTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "参数错误",
			Data:  nil,
		})
		return
	}

	// 检查配置是否需要重新初始化
	if configUpdated {
		log.GetLogger().Info("检测到配置更新，重新初始化服务")
		h.Service = service.NewService()
		configUpdated = false
	}

	svc := h.Service
	data, err := svc.GetTaskStatus(req)
	if err != nil {
		response.R(c, response.Response{
			Error: -1,
			Msg:   err.Error(),
			Data:  nil,
		})
		return
	}
	response.R(c, response.Response{
		Error: 0,
		Msg:   "成功",
		Data:  data,
	})
}

func (h Handler) GetTaskHistory(c *gin.Context) {
	tasks, err := storage.GetTaskHistory(200) // Increased limit for frontend pagination
	if err != nil {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "获取历史记录失败: " + err.Error(),
			Data:  nil,
		})
		return
	}

	// Convert to DTO if needed, or return raw list.
	// Returning raw list is fine as types.SubtitleTask matches JSON requirements
	response.R(c, response.Response{
		Error: 0,
		Msg:   "成功",
		Data:  tasks,
	})
}

func (h Handler) DeleteTask(c *gin.Context) {
	taskId := c.Param("taskId")
	if taskId == "" {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "taskId不能为空",
			Data:  nil,
		})
		return
	}

	// 1. Delete files from disk
	for _, taskPath := range taskDirCandidates(taskId) {
		if err := os.RemoveAll(taskPath); err != nil {
			log.GetLogger().Error("DeleteTask RemoveAll err", zap.String("path", taskPath), zap.Error(err))
			// Continue to delete from DB even if file deletion fails
		}
	}

	// 2. Delete from DB
	if err := storage.DeleteTask(taskId); err != nil {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "删除记录失败: " + err.Error(),
			Data:  nil,
		})
		return
	}

	response.R(c, response.Response{
		Error: 0,
		Msg:   "删除成功",
		Data:  nil,
	})
}

// RetryTask restarts a failed task by re-submitting it
func (h Handler) RetryTask(c *gin.Context) {
	taskId := c.Param("taskId")
	if taskId == "" {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "taskId不能为空",
			Data:  nil,
		})
		return
	}

	// Get the original task
	task, err := storage.GetTask(taskId)
	if err != nil {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "获取任务失败: " + err.Error(),
			Data:  nil,
		})
		return
	}

	// Allow retry of failed tasks (status=3) and completed tasks (status=2) for regeneration
	if task.Status != 3 && task.Status != 2 {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "只能重试失败或已完成的任务",
			Data:  nil,
		})
		return
	}

	// Resume/Retry logic: Do NOT delete files or DB record to allow resume capability to work

	// Determine voice code: use persisted one, or default if empty (legacy tasks)
	voiceCode := task.TtsVoiceCode
	if voiceCode == "" {
		voiceCode = "zh_female_wanqudashu_moon_bigtts" // Default safe Doubao voice (V3)
	}

	// Create request for retry, preserving original config as much as possible
	// Note: EmbedSubtitleVideoType and Tts are not persisted in DB, so we default to enabling video for retries
	// to avoid missing files.
	req := dto.StartVideoSubtitleTaskReq{
		Url:                    task.VideoSrc,
		ReuseTaskId:            task.TaskId,
		OriginLanguage:         string(task.OriginLanguage),
		TargetLang:             string(task.TargetLanguage),
		EmbedSubtitleVideoType: "all", // Force enable video generation (adaptive horizontal/vertical)
		Bilingual:              1,     // Default to Bilingual Yes
		Tts:                    1,     // Force Enable TTS for retry
		TtsVoiceCode:           voiceCode,
	}

	svc := h.Service
	data, err := svc.StartSubtitleTask(req)
	if err != nil {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "重试任务失败: " + err.Error(),
			Data:  nil,
		})
		return
	}

	response.R(c, response.Response{
		Error: 0,
		Msg:   "任务已重新提交",
		Data:  data,
	})
}

func (h Handler) UploadFile(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "未能获取文件",
			Data:  nil,
		})
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "未上传任何文件",
			Data:  nil,
		})
		return
	}

	// 保存每个文件
	var savedFiles []string
	uploadRoot := preferredUploadRoot()
	if err := os.MkdirAll(uploadRoot, 0o755); err != nil {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "创建上传目录失败",
			Data:  nil,
		})
		return
	}

	for _, file := range files {
		fileName := filepath.Base(file.Filename)
		savePath := filepath.Join(uploadRoot, fileName)
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			response.R(c, response.Response{
				Error: -1,
				Msg:   "文件保存失败: " + file.Filename,
				Data:  nil,
			})
			return
		}
		savedFiles = append(savedFiles, "local:"+savePath)
	}

	response.R(c, response.Response{
		Error: 0,
		Msg:   "文件上传成功",
		Data:  gin.H{"file_path": savedFiles},
	})
}

func (h Handler) DownloadFile(c *gin.Context) {
	requestedFile := c.Param("filepath")
	if requestedFile == "" {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "文件路径为空",
			Data:  nil,
		})
		return
	}

	// Only allow downloads from a small set of safe directories.
	// The router uses a wildcard (*filepath), so the param can contain slashes.
	requestedFile = strings.TrimPrefix(requestedFile, "/")
	requestedFile = strings.TrimPrefix(requestedFile, string(filepath.Separator))

	localFilePath, ok := resolveDownloadPath(requestedFile)
	if !ok {
		c.JSON(403, response.Response{Error: -1, Msg: "非法路径", Data: nil})
		return
	}

	fileInfo, err := os.Stat(localFilePath)
	if os.IsNotExist(err) {
		c.JSON(404, response.Response{Error: -1, Msg: "文件不存在", Data: nil})
		return
	}
	if err != nil {
		c.JSON(500, response.Response{Error: -1, Msg: "文件读取失败", Data: nil})
		return
	}
	if fileInfo.IsDir() {
		c.JSON(404, response.Response{Error: -1, Msg: "文件不存在", Data: nil})
		return
	}

	c.FileAttachment(localFilePath, filepath.Base(localFilePath))
}
