package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"krillin-ai/pkg/util"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s Service) embedSubtitles(ctx context.Context, stepParam *types.SubtitleTaskStepParam) error {
	var err error
	log.GetLogger().Info("embedSubtitles start", 
		zap.String("VideoType", stepParam.EmbedSubtitleVideoType),
		zap.Bool("EnableTts", stepParam.EnableTts),
		zap.String("VideoWithTtsFilePath", stepParam.VideoWithTtsFilePath))

	if stepParam.EmbedSubtitleVideoType == "horizontal" || stepParam.EmbedSubtitleVideoType == "vertical" || stepParam.EmbedSubtitleVideoType == "all" {
		var width, height int
		width, height, err = getResolution(stepParam.InputVideoPath)
		if err != nil {
			log.GetLogger().Error("embedSubtitles getResolution error", zap.Any("step param", stepParam), zap.Error(err))
			return fmt.Errorf("embedSubtitles getResolution error: %w", err)
		}

		// 横屏可以合成竖屏的，但竖屏暂时不支持合成横屏的
		if stepParam.EmbedSubtitleVideoType == "horizontal" || stepParam.EmbedSubtitleVideoType == "all" {
			if width < height {
				log.GetLogger().Info("检测到输入视频是竖屏，无法合成横屏视频，跳过")
			} else {
				log.GetLogger().Info("合成视频：横屏")
				err = embedSubtitles(stepParam, true, stepParam.EnableTts)
				if err != nil {
					log.GetLogger().Error("embedSubtitles embedSubtitles error", zap.Any("step param", stepParam), zap.Error(err))
					return fmt.Errorf("embedSubtitles embedSubtitles error: %w", err)
				}
				stepParam.SubtitleInfos = append(stepParam.SubtitleInfos, types.SubtitleFileInfo{
					Name: "horizontal_embed.mp4",
					Path: filepath.Join(stepParam.TaskBasePath, "output", types.SubtitleTaskHorizontalEmbedVideoFileName),
				})
			}
		}
		if stepParam.EmbedSubtitleVideoType == "vertical" || stepParam.EmbedSubtitleVideoType == "all" {
			if width > height {
				// 生成竖屏视频
				transferredVerticalVideoPath := filepath.Join(stepParam.TaskBasePath, types.SubtitleTaskTransferredVerticalVideoFileName)
				err = convertToVertical(stepParam.InputVideoPath, transferredVerticalVideoPath, stepParam.VerticalVideoMajorTitle, stepParam.VerticalVideoMinorTitle)
				if err != nil {
					log.GetLogger().Error("embedSubtitles convertToVertical error", zap.Any("step param", stepParam), zap.Error(err))
					return fmt.Errorf("embedSubtitles convertToVertical error: %w", err)
				}
				stepParam.InputVideoPath = transferredVerticalVideoPath
			}
			log.GetLogger().Info("合成视频：竖屏")
			err = embedSubtitles(stepParam, false, stepParam.EnableTts)
			if err != nil {
				log.GetLogger().Error("embedSubtitles embedSubtitles error", zap.Any("step param", stepParam), zap.Error(err))
				return fmt.Errorf("embedSubtitles embedSubtitles error: %w", err)
			}
			stepParam.SubtitleInfos = append(stepParam.SubtitleInfos, types.SubtitleFileInfo{
				Name: "vertical_embed.mp4",
				Path: filepath.Join(stepParam.TaskBasePath, "output", types.SubtitleTaskVerticalEmbedVideoFileName),
			})
		}
		log.GetLogger().Info("字幕嵌入视频成功")
		return nil
	}
	log.GetLogger().Info("合成视频：不合成")
	return nil
}

func splitMajorTextInHorizontal(text string, language types.StandardLanguageCode, maxWordOneLine int) []string {
	// 按语言情况分割
	var (
		segments []string
		sep      string
	)
	if language == types.LanguageNameSimplifiedChinese || language == types.LanguageNameTraditionalChinese ||
		language == types.LanguageNameJapanese || language == types.LanguageNameKorean || language == types.LanguageNameThai {
		segments = regexp.MustCompile(`.`).FindAllString(text, -1)
		sep = ""
	} else {
		segments = strings.Split(text, " ")
		sep = " "
	}

	totalWidth := len(segments)

	// 直接返回原句子
	if totalWidth <= maxWordOneLine {
		return []string{text}
	}

	// 确定拆分点，按2/5和3/5的比例拆分
	line1MaxWidth := int(float64(totalWidth) * 2 / 5)
	currentWidth := 0
	splitIndex := 0

	for i := range segments {
		currentWidth++

		// 当达到 2/5 宽度时，设置拆分点
		if currentWidth >= line1MaxWidth {
			splitIndex = i + 1
			break
		}
	}

	// 分割文本，保留原有句子格式

	line1 := util.CleanPunction(strings.Join(segments[:splitIndex], sep))
	line2 := util.CleanPunction(strings.Join(segments[splitIndex:], sep))

	return []string{line1, line2}
}

func splitChineseText(text string, maxWordLine int) []string {
	var lines []string
	words := []rune(text)
	for i := 0; i < len(words); i += maxWordLine {
		end := i + maxWordLine
		if end > len(words) {
			end = len(words)
		}
		lines = append(lines, string(words[i:end]))
	}
	return lines
}

func parseSrtTime(timeStr string) (time.Duration, error) {
	timeStr = strings.Replace(timeStr, ",", ".", 1)
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("parseSrtTime invalid time format: %s", timeStr)
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	secondsAndMilliseconds := strings.Split(parts[2], ".")
	if len(secondsAndMilliseconds) != 2 {
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}
	seconds, err := strconv.Atoi(secondsAndMilliseconds[0])
	if err != nil {
		return 0, err
	}
	milliseconds, err := strconv.Atoi(secondsAndMilliseconds[1])
	if err != nil {
		return 0, err
	}

	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(milliseconds)*time.Millisecond

	return duration, nil
}

func formatTimestamp(t time.Duration) string {
	hours := int(t.Hours())
	minutes := int(t.Minutes()) % 60
	seconds := int(t.Seconds()) % 60
	milliseconds := int(t.Milliseconds()) % 1000 / 10
	return fmt.Sprintf("%02d:%02d:%02d.%02d", hours, minutes, seconds, milliseconds)
}

func srtToAss(inputSRT, outputASS string, isHorizontal bool, stepParam *types.SubtitleTaskStepParam) error {
	file, err := os.Open(inputSRT)
	if err != nil {
		log.GetLogger().Error("srtToAss Open input srt error", zap.Error(err))
		return fmt.Errorf("srtToAss Open input srt error: %w", err)
	}
	defer file.Close()

	assFile, err := os.Create(outputASS)
	if err != nil {
		log.GetLogger().Error("srtToAss Create output ass error", zap.Error(err))
		return fmt.Errorf("srtToAss Create output ass error: %w", err)
	}
	defer assFile.Close()
	scanner := bufio.NewScanner(file)

	if isHorizontal {
		_, _ = assFile.WriteString(types.AssHeaderHorizontal)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			// 读取时间戳行
			if !scanner.Scan() {
				break
			}
			timestampLine := scanner.Text()
			parts := strings.Split(timestampLine, " --> ")
			if len(parts) != 2 {
				continue // 无效时间戳格式
			}

			startTimeStr := strings.TrimSpace(parts[0])
			endTimeStr := strings.TrimSpace(parts[1])
			startTime, err := parseSrtTime(startTimeStr)
			if err != nil {
				log.GetLogger().Error("srtToAss parseSrtTime error", zap.Error(err))
				return fmt.Errorf("srtToAss parseSrtTime error: %w", err)
			}
			endTime, err := parseSrtTime(endTimeStr)
			if err != nil {
				log.GetLogger().Error("srtToAss parseSrtTime error", zap.Error(err))
				return fmt.Errorf("srtToAss parseSrtTime error: %w", err)
			}

			var subtitleLines []string
			for scanner.Scan() {
				textLine := scanner.Text()
				if textLine == "" {
					break // 字幕块结束
				}
				subtitleLines = append(subtitleLines, textLine)
			}

			if len(subtitleLines) < 2 {
				continue
			}

			// ASS条目 - 目标语言（中文）在上且大（Major），原语言（英文）在下且小（Minor）
			startFormatted := formatTimestamp(startTime)
			endFormatted := formatTimestamp(endTime)
			// subtitleLines[0] 是第一行，subtitleLines[1] 是第二行
			// 根据 SubtitleResultType 判断哪个是目标语言
			var majorText, minorText string
			if stepParam.SubtitleResultType == types.SubtitleResultTypeBilingualTranslationOnTop {
				// 目标语言在上（subtitleLines[0]），原语言在下（subtitleLines[1]）
				majorText = subtitleLines[0]  // 中文
				minorText = util.CleanPunction(subtitleLines[1])  // 英文
			} else {
				// 原语言在上（subtitleLines[0]），目标语言在下（subtitleLines[1]） - 需要交换
				majorText = subtitleLines[1]  // 中文
				minorText = util.CleanPunction(subtitleLines[0])  // 英文
			}
			combinedText := fmt.Sprintf("{\\an2}{\\rMajor}%s\\N{\\rMinor}%s", majorText, minorText)
			_, _ = assFile.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Major,,0,0,0,,%s\n", startFormatted, endFormatted, combinedText))
		}
	} else {
		// TODO 竖屏拆分调优
		_, _ = assFile.WriteString(types.AssHeaderVertical)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			// 读取时间戳行
			if !scanner.Scan() {
				break
			}
			timestampLine := scanner.Text()
			parts := strings.Split(timestampLine, " --> ")
			if len(parts) != 2 {
				continue // 无效时间戳格式
			}

			startTimeStr := strings.TrimSpace(parts[0])
			endTimeStr := strings.TrimSpace(parts[1])
			startTime, err := parseSrtTime(startTimeStr)
			if err != nil {
				log.GetLogger().Error("srtToAss parseSrtTime error", zap.Error(err))
				return fmt.Errorf("srtToAss parseSrtTime error: %w", err)
			}
			endTime, err := parseSrtTime(endTimeStr)
			if err != nil {
				log.GetLogger().Error("srtToAss parseSrtTime error", zap.Error(err))
				return fmt.Errorf("srtToAss parseSrtTime error: %w", err)
			}

			// 读取内容行 (修复: 支持多行内容，如双语字幕)
			var subtitleLines []string
			for scanner.Scan() {
				textLine := scanner.Text()
				if textLine == "" {
					break // 字幕块结束
				}
				subtitleLines = append(subtitleLines, textLine)
			}

			if len(subtitleLines) == 0 {
				continue
			}

			startFormatted := formatTimestamp(startTime)
			endFormatted := formatTimestamp(endTime)

			// 双语字幕处理 (显示两行)
			if len(subtitleLines) >= 2 {
				var majorText, minorText string
				if stepParam.SubtitleResultType == types.SubtitleResultTypeBilingualTranslationOnTop {
					majorText = subtitleLines[0]
					minorText = util.CleanPunction(subtitleLines[1])
				} else {
					majorText = subtitleLines[1]
					minorText = util.CleanPunction(subtitleLines[0])
				}
				// 竖屏双语：直接显示，不做时间切分，防止错位
				combinedText := fmt.Sprintf("{\\an2}{\\rMajor}%s\\N{\\rMinor}%s", majorText, minorText)
				_, _ = assFile.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Major,,0,0,0,,%s\n", startFormatted, endFormatted, combinedText))
				continue
			}

			// 单语字幕处理 (保持原有逻辑)
			content := subtitleLines[0]
			totalTime := endTime - startTime

			if !util.ContainsAlphabetic(content) {
				// 处理中文字幕
				chineseLines := splitChineseText(content, 10)
				for i, line := range chineseLines {
					iStart := startTime + time.Duration(float64(i)*float64(totalTime)/float64(len(chineseLines)))
					iEnd := startTime + time.Duration(float64(i+1)*float64(totalTime)/float64(len(chineseLines)))
					if iEnd > endTime {
						iEnd = endTime
					}

					startFormatted := formatTimestamp(iStart)
					endFormatted := formatTimestamp(iEnd)
					cleanedText := util.CleanPunction(line)
					combinedText := fmt.Sprintf("{\\an2}{\\rMajor}%s", cleanedText)
					_, _ = assFile.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Major,,0,0,0,,%s\n", startFormatted, endFormatted, combinedText))
				}
			} else {
				// 处理英文字幕
				cleanedText := util.CleanPunction(content)
				combinedText := fmt.Sprintf("{\\an2}{\\rMinor}%s", cleanedText)
				_, _ = assFile.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Minor,,0,0,0,,%s\n", startFormatted, endFormatted, combinedText))
			}
		}
	}
	return nil
}

func embedSubtitles(stepParam *types.SubtitleTaskStepParam, isHorizontal bool, withTts bool) error {
	outputFileName := types.SubtitleTaskVerticalEmbedVideoFileName
	if isHorizontal {
		outputFileName = types.SubtitleTaskHorizontalEmbedVideoFileName
	}
	assPath := filepath.Join(stepParam.TaskBasePath, "formatted_subtitles.ass")

	if err := srtToAss(stepParam.BilingualSrtFilePath, assPath, isHorizontal, stepParam); err != nil {
		log.GetLogger().Error("embedSubtitles srtToAss error", zap.Any("step param", stepParam), zap.Error(err))
		return fmt.Errorf("embedSubtitles srtToAss error: %w", err)
	}
	input := stepParam.InputVideoPath
	if withTts {
		input = stepParam.VideoWithTtsFilePath
	}


	// Escape path properly for FFmpeg ass filter
	// Use absolute path and escape special characters including apostrophes
	assPathEscaped := strings.ReplaceAll(assPath, "'", "\\'")
	assPathEscaped = strings.ReplaceAll(assPathEscaped, "\\", "/")
	
	cmd := exec.Command(storage.FfmpegPath, "-y", "-i", input, "-vf", fmt.Sprintf("ass='%s'", assPathEscaped), "-c:a", "aac", "-b:a", "192k", filepath.Join(stepParam.TaskBasePath, fmt.Sprintf("/output/%s", outputFileName)))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.GetLogger().Error("embedSubtitles embed subtitle into video ffmpeg error", zap.String("video path", stepParam.InputVideoPath), zap.String("output", string(output)), zap.Error(err))
		return fmt.Errorf("embedSubtitles embed subtitle into video ffmpeg error: %w", err)
	}
	return nil
}

func getFontPaths() (string, string, error) {
	switch runtime.GOOS {
	case "windows":
		return "C\\:/Windows/Fonts/msyhbd.ttc", "C\\:/Windows/Fonts/msyh.ttc", nil // 在ffmpeg参数里必须这样写
	case "darwin":
		return "/System/Library/Fonts/Supplemental/Arial Bold.ttf", "/System/Library/Fonts/Supplemental/Arial.ttf", nil
	case "linux":
		return "/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc", "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc", nil
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func getResolution(inputVideo string) (int, int, error) {
	// 获取视频信息
	cmdArgs := []string{
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		inputVideo,
	}
	cmd := exec.Command(storage.FfprobePath, cmdArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		log.GetLogger().Error("获取视频分辨率失败", zap.String("output", out.String()), zap.Error(err))
		return 0, 0, err
	}

	output := strings.TrimSpace(out.String())
	output = strings.TrimSuffix(output, "x") // 去除尾部可能存在的x,例如1920x1080x

	re := regexp.MustCompile(`^(\d+)x(\d+)$`)
	dimensions := re.FindStringSubmatch(output)
	if len(dimensions) != 3 {
		log.GetLogger().Error("获取视频分辨率失败", zap.String("output", output))
		return 0, 0, fmt.Errorf("invalid resolution format: %s", output)
	}

	width, _ := strconv.Atoi(dimensions[1])
	height, _ := strconv.Atoi(dimensions[2])
	return width, height, nil
}

func convertToVertical(inputVideo, outputVideo, majorTitle, minorTitle string) error {
	if _, err := os.Stat(outputVideo); err == nil {
		log.GetLogger().Info("竖屏视频已存在", zap.String("outputVideo", outputVideo))
		return nil
	}

	fontBold, fontRegular, err := getFontPaths()
	if err != nil {
		log.GetLogger().Error("获取字体路径失败", zap.Error(err))
		return err
	}

	cmdArgs := []string{
		"-i", inputVideo,
		"-vf", fmt.Sprintf("scale=720:1280:force_original_aspect_ratio=decrease,pad=720:1280:(ow-iw)/2:(oh-ih)*2/5,drawbox=y=0:h=100:c=black@1:t=fill,drawtext=text='%s':x=(w-text_w)/2:y=210:fontsize=55:fontcolor=yellow:box=1:boxcolor=black@0.5:fontfile='%s',drawtext=text='%s':x=(w-text_w)/2:y=280:fontsize=40:fontcolor=yellow:box=1:boxcolor=black@0.5:fontfile='%s'",
			majorTitle, fontBold, minorTitle, fontRegular),
		"-r", "30",
		"-b:v", "7587k",
		"-c:a", "aac",
		"-b:a", "192k",
		"-c:v", "libx264",
		"-preset", "fast",
		"-y",
		outputVideo,
	}
	cmd := exec.Command(storage.FfmpegPath, cmdArgs...)
	var output []byte
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.GetLogger().Error("视频转竖屏失败", zap.String("output", string(output)), zap.Error(err))
		return err
	}

	fmt.Printf("竖屏视频已保存到: %s\n", outputVideo)
	return nil
}
