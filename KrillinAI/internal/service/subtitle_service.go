package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"krillin-ai/internal/dto"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	apperrors "krillin-ai/pkg/errors"
	"krillin-ai/pkg/util"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func (s Service) StartSubtitleTask(req dto.StartVideoSubtitleTaskReq) (*dto.StartVideoSubtitleTaskResData, error) {
	// 校验链接
	if strings.Contains(req.Url, "youtube.com") {
		videoId, _ := util.GetYouTubeID(req.Url)
		if videoId == "" {
			return nil, apperrors.New(apperrors.CodeUnsupportedURL, "YouTube链接不合法 Invalid YouTube URL")
		}
	}
	if strings.Contains(req.Url, "bilibili.com") {
		videoId := util.GetBilibiliVideoId(req.Url)
		if videoId == "" {
			return nil, apperrors.New(apperrors.CodeUnsupportedURL, "Bilibili链接不合法 Invalid Bilibili URL")
		}
	}
	// 生成或复用任务id
	var taskId string
	if req.ReuseTaskId != "" {
		taskId = req.ReuseTaskId
	} else {
		seperates := strings.Split(req.Url, "/")
		taskId = fmt.Sprintf("%s_%s", util.SanitizePathName(string([]rune(strings.ReplaceAll(seperates[len(seperates)-1], " ", ""))[:16])), util.GenerateRandStringWithUpperLowerNum(4))
		taskId = strings.ReplaceAll(taskId, "=", "") // 等于号影响ffmpeg处理
		taskId = strings.ReplaceAll(taskId, "?", "") // 问号影响ffmpeg处理
	}
	// 构造任务所需参数
	var resultType types.SubtitleResultType
	// 根据入参选项确定要返回的字幕类型
	if req.TargetLang == "none" {
		resultType = types.SubtitleResultTypeOriginOnly
	} else {
		if req.Bilingual == types.SubtitleTaskBilingualYes {
			if req.TranslationSubtitlePos == types.SubtitleTaskTranslationSubtitlePosTop {
				resultType = types.SubtitleResultTypeBilingualTranslationOnTop
			} else {
				resultType = types.SubtitleResultTypeBilingualTranslationOnBottom
			}
		} else {
			resultType = types.SubtitleResultTypeTargetOnly
		}
	}
	// 文字替换map
	replaceWordsMap := make(map[string]string)
	if len(req.Replace) > 0 {
		for _, replace := range req.Replace {
			beforeAfter := strings.Split(replace, "|")
			if len(beforeAfter) == 2 {
				replaceWordsMap[beforeAfter[0]] = beforeAfter[1]
			} else {
				log.GetLogger().Info("generateAudioSubtitles replace param length err", zap.Any("replace", replace), zap.Any("taskId", taskId))
			}
		}
	}
	var err error
	ctx := context.Background()
	// 创建字幕任务文件夹
	taskBasePath := filepath.Join("./tasks", taskId)
	if _, err = os.Stat(taskBasePath); os.IsNotExist(err) {
		// 不存在则创建
		err = os.MkdirAll(filepath.Join(taskBasePath, "output"), os.ModePerm)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask MkdirAll err", zap.Any("req", req), zap.Error(err))
		}
	}

	// 创建或更新任务
	var taskPtr *types.SubtitleTask
	if req.ReuseTaskId != "" {
		taskPtr, _ = storage.GetTask(taskId)
	}

	if taskPtr == nil {
		taskPtr = &types.SubtitleTask{
			TaskId:       taskId,
			VideoSrc:     req.Url,
			Status:       types.SubtitleTaskStatusProcessing,
			TtsVoiceCode: req.TtsVoiceCode, // New: Persist voice code
		}
	} else {
		// Reset status for retry
		taskPtr.Status = types.SubtitleTaskStatusProcessing
		taskPtr.FailReason = ""
		taskPtr.StatusMsg = "正在重试 Retrying..."
		// Update VideoSrc just in case
		taskPtr.VideoSrc = req.Url
		if req.TtsVoiceCode != "" {
			taskPtr.TtsVoiceCode = req.TtsVoiceCode // Update voice code if provided
		}
	}
	// Migrate to DB: storage.SubtitleTasks.Store(taskId, taskPtr) -> SaveTask
	if err := storage.SaveTask(taskPtr); err != nil {
		log.GetLogger().Error("StartVideoSubtitleTask SaveTask err", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.CodeDBError, "保存任务失败 Failed to save task", err)
	}

	// 处理声音克隆源（火山：上传参考音频训练 speaker_id，合成时 voice_type=SpeakerID 且 cluster=volcano_icl）
	// 约束：火山的 upload/status 接口要求 speaker_id (S_*) 从控制台获取，因此这里复用 req.TtsVoiceCode 作为 SpeakerID。
	if req.TtsVoiceCloneSrcFileUrl != "" {
		localFile := strings.TrimPrefix(req.TtsVoiceCloneSrcFileUrl, "local:")
		if req.TtsVoiceCode == "" {
			return nil, errors.New("启用语音克隆时必须填写 voice_code（SpeakerID，形如 S_***）")
		}
		if _, err := s.prepareVolcVoiceClone(ctx, localFile, req.TtsVoiceCode); err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask voice clone failed", zap.Any("req", req), zap.Error(err))
			return nil, fmt.Errorf("语音克隆训练失败: %w", err)
		}
	}

	stepParam := &types.SubtitleTaskStepParam{
		TaskId:                  taskId,
		TaskPtr:                 taskPtr,
		TaskBasePath:            taskBasePath,
		Link:                    req.Url,
		AudioDownloadUrl:        req.AudioUrl, // Pass separate audio URL
		SubtitleResultType:      resultType,
		EnableModalFilter:       req.ModalFilter == types.SubtitleTaskModalFilterYes,
		EnableTts:               req.Tts == types.SubtitleTaskTtsYes,
		TtsVoiceCode:            req.TtsVoiceCode,
		ReplaceWordsMap:         replaceWordsMap,
		OriginLanguage:          types.StandardLanguageCode(req.OriginLanguage),
		TargetLanguage:          types.StandardLanguageCode(req.TargetLang),
		UserUILanguage:          types.StandardLanguageCode(req.Language),
		EmbedSubtitleVideoType:  req.EmbedSubtitleVideoType,
		VerticalVideoMajorTitle: req.VerticalMajorTitle,
		VerticalVideoMinorTitle: req.VerticalMinorTitle,
		MaxWordOneLine:          12, // 默认值
	}
	log.GetLogger().Info("StartVideoSubtitleTask stepParam initialized",
		zap.Bool("EnableTts", stepParam.EnableTts),
		zap.Any("SubtitleResultType", stepParam.SubtitleResultType),
		zap.String("EmbedSubtitleVideoType", stepParam.EmbedSubtitleVideoType))
	if req.OriginLanguageWordOneLine != 0 {
		stepParam.MaxWordOneLine = req.OriginLanguageWordOneLine
	}

	log.GetLogger().Info("current task info", zap.String("taskId", taskId), zap.Any("param", stepParam))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				log.GetLogger().Error("autoVideoSubtitle panic", zap.Any("panic:", r), zap.Any("stack:", buf))
				stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
				_ = storage.SaveTask(stepParam.TaskPtr) // Persist failure
			}
		}()
		// 新版流程：链接->本地音频文件->视频信息获取（若有）->本地字幕文件->语言合成->视频合成->字幕文件链接生成
		log.GetLogger().Info("video subtitle start task", zap.String("taskId", taskId))
		stepParam.TaskPtr.StatusMsg = "正在下载资源 Downloading Resources..."
		_ = storage.SaveTask(stepParam.TaskPtr)

		err = s.linkToFile(ctx, stepParam)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask linkToFile err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			stepParam.TaskPtr.StatusMsg = "下载失败 Download Failed"
			_ = storage.SaveTask(stepParam.TaskPtr)
			return
		}

		// 获取视频信息（标题、封面、简介总结）
		stepParam.TaskPtr.StatusMsg = "正在分析视频信息 Analyzing Video Info..."
		_ = storage.SaveTask(stepParam.TaskPtr)

		err = s.getVideoInfo(ctx, stepParam)
		_ = storage.SaveTask(stepParam.TaskPtr)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask getVideoInfo err", zap.Any("req", req), zap.Error(err))
		}

		stepParam.TaskPtr.StatusMsg = "正在转录与翻译 Transcribing & Translating..."
		_ = storage.SaveTask(stepParam.TaskPtr)

		err = s.audioToSubtitle(ctx, stepParam)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask audioToSubtitle err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			stepParam.TaskPtr.StatusMsg = "转录翻译失败 Transcription/Translation Failed"
			_ = storage.SaveTask(stepParam.TaskPtr)
			return
		}

		// New: Ensure summary is generated even for non-Youtube sources
		s.generateSummaryIfMissing(ctx, stepParam)

		if req.Tts == types.SubtitleTaskTtsYes {
			stepParam.TaskPtr.StatusMsg = "正在生成配音 Generating Dubbing..."
			_ = storage.SaveTask(stepParam.TaskPtr)
		}

		err = s.srtFileToSpeech(ctx, stepParam)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask srtFileToSpeech err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			stepParam.TaskPtr.StatusMsg = "配音生成失败 Dubbing Failed"
			_ = storage.SaveTask(stepParam.TaskPtr)
			return
		}

		if req.EmbedSubtitleVideoType != "" && req.EmbedSubtitleVideoType != "none" {
			stepParam.TaskPtr.StatusMsg = "正在合成视频 Compositing Video..."
			_ = storage.SaveTask(stepParam.TaskPtr)
		}

		err = s.embedSubtitles(ctx, stepParam)
		_ = storage.SaveTask(stepParam.TaskPtr)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask embedSubtitles err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			stepParam.TaskPtr.StatusMsg = "视频合成失败 Video Composition Failed"
			// SubtitleInfos will be persisted automatically via GORM relationship when SaveTask is called
			_ = storage.SaveTask(stepParam.TaskPtr)
			return
		}

		stepParam.TaskPtr.StatusMsg = "正在完成 Finalizing..."
		_ = storage.SaveTask(stepParam.TaskPtr)

		err = s.uploadSubtitles(ctx, stepParam)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask uploadSubtitles err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			stepParam.TaskPtr.StatusMsg = "结果处理失败 Final Processing Failed"
			_ = storage.SaveTask(stepParam.TaskPtr)
			return
		}

		stepParam.TaskPtr.Status = types.SubtitleTaskStatusSuccess
		stepParam.TaskPtr.StatusMsg = "任务完成 Completed"
		_ = storage.SaveTask(stepParam.TaskPtr)

		log.GetLogger().Info("video subtitle task end", zap.String("taskId", taskId))
	}()

	return &dto.StartVideoSubtitleTaskResData{
		TaskId: taskId,
	}, nil
}

func (s Service) GetTaskStatus(req dto.GetVideoSubtitleTaskReq) (*dto.GetVideoSubtitleTaskResData, error) {
	taskPtr, err := storage.GetTask(req.TaskId)
	if err != nil {
		return nil, errors.New("任务不存在或查询失败")
	}
	if taskPtr.Status == types.SubtitleTaskStatusFailed {
		return nil, fmt.Errorf("任务失败，原因：%s", taskPtr.FailReason)
	}
	return &dto.GetVideoSubtitleTaskResData{
		TaskId:         taskPtr.TaskId,
		ProcessPercent: taskPtr.ProcessPct,
		VideoInfo: &dto.VideoInfo{
			Title:                 taskPtr.Title,
			Description:           taskPtr.Description,
			TranslatedTitle:       taskPtr.TranslatedTitle,
			TranslatedDescription: taskPtr.TranslatedDescription,
		},
		SubtitleInfo: lo.Map(taskPtr.SubtitleInfos, func(item types.SubtitleInfo, _ int) *dto.SubtitleInfo {
			return &dto.SubtitleInfo{
				Name:        item.Name,
				DownloadUrl: item.DownloadUrl,
			}
		}),
		TargetLanguage:    taskPtr.TargetLanguage,
		SpeechDownloadUrl: taskPtr.SpeechDownloadUrl,
	}, nil
}
