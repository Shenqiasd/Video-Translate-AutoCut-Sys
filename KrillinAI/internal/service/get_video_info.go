package service

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"krillin-ai/config"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"os"
	"path/filepath"
	"os/exec"
	"strings"
)

func (s Service) getVideoInfo(ctx context.Context, stepParam *types.SubtitleTaskStepParam) error {
	link := stepParam.Link
	if strings.Contains(link, "youtube.com") || strings.Contains(link, "bilibili.com") {
		var (
			err                error
			title, description string
		)
		// 1. 获取标题和描述
		titleCmdArgs := []string{"--skip-download", "--encoding", "utf-8", "--get-title", stepParam.Link}
		descriptionCmdArgs := []string{"--skip-download", "--encoding", "utf-8", "--get-description", stepParam.Link}
		// 2. 下载封面图
		// --write-thumbnail: 下载封面
		// --skip-download: 不下载视频
		// --convert-thumbnails jpg: 转换为jpg格式
		// -o: 指定输出路径
		coverPath := filepath.Join(stepParam.TaskBasePath, "output", "cover") // yt-dlp 会自动添加后缀
		thumbnailCmdArgs := []string{"--skip-download", "--write-thumbnail", "--convert-thumbnails", "jpg", "-o", coverPath, stepParam.Link}

		// 公共参数配置
		commonArgs := []string{"--cookies", "/app/cookies.txt"}
		if config.Conf.App.Proxy != "" {
			commonArgs = append(commonArgs, "--proxy", config.Conf.App.Proxy)
		}
		if storage.FfmpegPath != "ffmpeg" {
			commonArgs = append(commonArgs, "--ffmpeg-location", storage.FfmpegPath)
		}

		titleCmdArgs = append(titleCmdArgs, commonArgs...)
		descriptionCmdArgs = append(descriptionCmdArgs, commonArgs...)
		thumbnailCmdArgs = append(thumbnailCmdArgs, commonArgs...)

		// 执行获取标题
		cmd := exec.Command(storage.YtdlpPath, titleCmdArgs...)
		var output []byte
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.GetLogger().Error("getVideoInfo yt-dlp get title error", zap.Any("stepParam", stepParam), zap.String("output", string(output)), zap.Error(err))
			// 继续尝试，不强制退出
		}
		title = strings.TrimSpace(string(output))

		// 执行获取描述
		cmd = exec.Command(storage.YtdlpPath, descriptionCmdArgs...)
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.GetLogger().Error("getVideoInfo yt-dlp get description error", zap.Any("stepParam", stepParam), zap.String("output", string(output)), zap.Error(err))
		}
		description = strings.TrimSpace(string(output))

		// 执行下载封面
		cmd = exec.Command(storage.YtdlpPath, thumbnailCmdArgs...)
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.GetLogger().Error("getVideoInfo yt-dlp download thumbnail error", zap.Any("stepParam", stepParam), zap.String("output", string(output)), zap.Error(err))
		} else {
			// 查找下载的封面文件
			matches, _ := filepath.Glob(filepath.Join(stepParam.TaskBasePath, "output", "cover.*"))
			if len(matches) > 0 {
				stepParam.TaskPtr.Cover = matches[0] // 保存封面路径
				stepParam.SubtitleInfos = append(stepParam.SubtitleInfos, types.SubtitleFileInfo{
					Name: "cover" + filepath.Ext(matches[0]),
					Path: matches[0],
				})
			}
		}

		log.GetLogger().Debug("getVideoInfo title and description", zap.String("title", title), zap.String("description", description))

		// 3. AI 总结与翻译
		var result string
		// 使用新的 SummarizePrompt
		result, err = s.ChatCompleter.ChatCompletion(fmt.Sprintf(types.SummaryAndTitlePrompt, title+"####"+description))
		if err != nil {
			log.GetLogger().Error("getVideoInfo openai chat completion error", zap.Any("stepParam", stepParam), zap.Error(err))
		}
		log.GetLogger().Debug("getVideoInfo translate video info result", zap.String("result", result))

		taskPtr := stepParam.TaskPtr
		taskPtr.Title = title
		taskPtr.Description = description
		taskPtr.OriginLanguage = string(stepParam.OriginLanguage)
		taskPtr.TargetLanguage = string(stepParam.TargetLanguage)
		taskPtr.ProcessPct = 10

		splitResult := strings.Split(result, "####")
		var summaryContent string
		if len(splitResult) >= 2 {
			taskPtr.TranslatedTitle = strings.TrimSpace(splitResult[0])
			taskPtr.TranslatedDescription = strings.TrimSpace(splitResult[1])
			summaryContent = fmt.Sprintf("标题：%s\n\n简介：\n%s", taskPtr.TranslatedTitle, taskPtr.TranslatedDescription)
		} else {
			taskPtr.TranslatedTitle = result
			summaryContent = result
			log.GetLogger().Error("getVideoInfo translate video info error split result format invalid", zap.Any("stepParam", stepParam), zap.Any("translate result", result))
		}

		// 4. 保存总结到 summary.txt
		summaryFile := filepath.Join(stepParam.TaskBasePath, "output", "summary.txt")
		if err := os.WriteFile(summaryFile, []byte(summaryContent), 0644); err != nil {
			log.GetLogger().Error("getVideoInfo save summary.txt error", zap.Error(err))
		} else {
			stepParam.SubtitleInfos = append(stepParam.SubtitleInfos, types.SubtitleFileInfo{
				Name: "summary.txt",
				Path: summaryFile,
			})
		}
	}
	return nil
}

// generateSummaryIfMissing 检查是否已有summary，如果没有则通过转录文本生成
func (s Service) generateSummaryIfMissing(ctx context.Context, stepParam *types.SubtitleTaskStepParam) {
	// 1. 检查是否存在summary
	hasSummary := false
	for _, info := range stepParam.SubtitleInfos {
		if strings.Contains(info.Name, "summary") {
			hasSummary = true
			break
		}
	}
	// Double check file system
	summaryFile := filepath.Join(stepParam.TaskBasePath, "output", "summary.txt")
	if !hasSummary {
		if _, err := os.Stat(summaryFile); err == nil {
			hasSummary = true
		}
	}

	if hasSummary {
		return
	}

	log.GetLogger().Info("Summary missing, generating from transcript...", zap.String("taskId", stepParam.TaskId))

	// 2. 读取转录文本
	transcriptPath := filepath.Join(stepParam.TaskBasePath, "output", types.SubtitleTaskOriginLanguageTextFileName)
	content, err := os.ReadFile(transcriptPath)
	if err != nil {
		log.GetLogger().Warn("generateSummaryIfMissing read transcript failed", zap.Error(err))
		return
	}

	text := string(content)
	if len(text) == 0 {
		return
	}

	// 3. 截断过长文本 (OpenAI Context limits)
	maxLength := 8000
	if len(text) > maxLength {
		text = text[:maxLength] + "..."
	}

	// 4. 调用LLM生成总结
	prompt := fmt.Sprintf(types.SummaryTranscriptPrompt, text)
	result, err := s.ChatCompleter.ChatCompletion(prompt)
	if err != nil {
		log.GetLogger().Error("generateSummaryIfMissing chat completion error", zap.Error(err))
		return
	}

	// 5. 解析并保存
	taskPtr := stepParam.TaskPtr
	splitResult := strings.Split(result, "####")
	var summaryContent string
	
	if len(splitResult) >= 2 {
		// 如果原有标题为空(非Youtube来源)，尝试根据AI生成的标题填充
		if taskPtr.TranslatedTitle == "" {
			taskPtr.TranslatedTitle = strings.TrimSpace(splitResult[0])
		}
		summaryContent = fmt.Sprintf("标题：%s\n\n简介：\n%s", strings.TrimSpace(splitResult[0]), strings.TrimSpace(splitResult[1]))
		// 也可以更新 Task Description
		if taskPtr.TranslatedDescription == "" {
			taskPtr.TranslatedDescription = strings.TrimSpace(splitResult[1])
		}
	} else {
		summaryContent = result
	}

	if err := os.WriteFile(summaryFile, []byte(summaryContent), 0644); err != nil {
		log.GetLogger().Error("generateSummaryIfMissing save summary.txt error", zap.Error(err))
	} else {
		log.GetLogger().Info("Summary generated from transcript successfully")
		stepParam.SubtitleInfos = append(stepParam.SubtitleInfos, types.SubtitleFileInfo{
			Name: "summary.txt",
			Path: summaryFile,
		})
	}
}
