package service

import (
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
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// 输入中文字幕，生成配音
func (s Service) srtFileToSpeech(ctx context.Context, stepParam *types.SubtitleTaskStepParam) error {
	if !stepParam.EnableTts {
		return nil
	}
	// Step 1: 解析字幕文件
	subtitles, err := parseSRT(stepParam.TtsSourceFilePath)
	if err != nil {
		log.GetLogger().Error("srtFileToSpeech parseSRT error", zap.Any("stepParam", stepParam), zap.Error(err))
		return fmt.Errorf("srtFileToSpeech parseSRT error: %w", err)
	}

	var audioFiles []string
	// Track the actual physical duration of the generated audio stream
	var currentAudioCursor float64 = 0
	
	// 创建文件记录音频的开始和结束时间
	durationDetailFile, err := os.Create(filepath.Join(stepParam.TaskBasePath, types.TtsAudioDurationDetailsFileName))
	if err != nil {
		log.GetLogger().Error("srtFileToSpeech create durationDetailFile error", zap.Any("stepParam", stepParam), zap.Error(err))
		return fmt.Errorf("srtFileToSpeech create durationDetailFile error: %w", err)
	}
	defer durationDetailFile.Close()

	for i, sub := range subtitles {
		// Parse start/end times
		startTime, err := time.Parse("15:04:05,000", sub.Start)
		if err != nil {
			log.GetLogger().Error("srtFileToSpeech parse start time error", zap.Error(err))
			return fmt.Errorf("srtFileToSpeech parse start time error: %w", err)
		}
		endTime, err := time.Parse("15:04:05,000", sub.End)
		if err != nil {
			log.GetLogger().Error("srtFileToSpeech parse end time error", zap.Error(err))
			return fmt.Errorf("srtFileToSpeech parse end time error: %w", err)
		}

		// Calculate start time in seconds from 00:00:00
		startSeconds := float64(startTime.Hour()*3600 + startTime.Minute()*60 + startTime.Second()) + float64(startTime.Nanosecond())/1e9
		
		// 1. Calculate and Fill Gap
		// Gap is the time between where we are now (cursor) and where the subtitle should start
		gapDuration := startSeconds - currentAudioCursor
		
		if gapDuration > 0.01 { // Only generate silence if gap is significant (>10ms)
			silenceFile := filepath.Join(stepParam.TaskBasePath, fmt.Sprintf("gap_silence_%d.wav", i))
			err := newGenerateSilence(silenceFile, gapDuration)
			if err != nil {
				return fmt.Errorf("generate gap silence error: %w", err)
			}
			audioFiles = append(audioFiles, silenceFile)
			
			// Update cursor
			silenceActualDur, _ := util.GetAudioDuration(silenceFile)
			currentAudioCursor += silenceActualDur
			
			// detailed logging
			durationDetailFile.WriteString(fmt.Sprintf("Silence Gap %d: duration=%.3f, new_cursor=%.3f\n", i, silenceActualDur, currentAudioCursor))
		}

		// 2. Process Subtitle Audio
		// Target duration is strictly the subtitle duration
		targetDuration := endTime.Sub(startTime).Seconds()
		
		// Handle short duration
		if targetDuration < 0.5 {
			targetDuration = 0.5
		}

		outputFile := filepath.Join(stepParam.TaskBasePath, fmt.Sprintf("subtitle_%d.wav", i+1))

		// 3. Generate TTS Audio
		// Call TTS Service to generate audio file
		err = s.TtsClient.Text2Speech(sub.Text, stepParam.TtsVoiceCode, outputFile)
		if err != nil {
			log.GetLogger().Error("srtFileToSpeech TTS generation error", 
				zap.Int("index", i+1),
				zap.String("text", sub.Text),
				zap.Error(err))
			// Don't return error immediately? The logic below handles missing files in processSubtitlesConcurrently (which this is NOT).
			// This is the sequential version.
			// If TTS fails here, we should probably return error or generate silence?
			// The original code seemed to rely on concurrency batching, but here it's sequential?
			// Wait, the code I read earlier had `processSubtitlesConcurrently` function...
			// But `srtFileToSpeech` (lines 45-103) is a sequential loop!
			// Did I overlook a call to `processSubtitlesConcurrently`?
			// No, `srtFileToSpeech` seems to be the main loop.
			// Let's assume sequential for now.
			return fmt.Errorf("TTS generation failed for subtitle %d: %w", i+1, err)
		}
		adjustedFile := filepath.Join(stepParam.TaskBasePath, fmt.Sprintf("adjusted_%d.wav", i+1))
		
		actualDuration, err := adjustAudioDuration(outputFile, adjustedFile, stepParam.TaskBasePath, targetDuration)
		if err != nil {
			log.GetLogger().Error("srtFileToSpeech adjustAudioDuration error", zap.Error(err))
			return fmt.Errorf("adjustAudioDuration error: %w", err)
		}

		audioFiles = append(audioFiles, adjustedFile)
		currentAudioCursor += actualDuration
		
		durationDetailFile.WriteString(fmt.Sprintf("Audio %d: target=%.3f, actual=%.3f, new_cursor=%.3f\n", i+1, targetDuration, actualDuration, currentAudioCursor))
	}
	
	// Step 6: 拼接所有音频文件
	finalOutput := filepath.Join(stepParam.TaskBasePath, types.TtsResultAudioFileName)
	err = concatenateAudioFiles(audioFiles, finalOutput, stepParam.TaskBasePath)
	if err != nil {
		log.GetLogger().Error("srtFileToSpeech concatenateAudioFiles error", zap.Any("stepParam", stepParam), zap.Error(err))
		return fmt.Errorf("srtFileToSpeech concatenateAudioFiles error: %w", err)
	}
	stepParam.TtsResultFilePath = finalOutput

	// Step 7: 音频分离与混音 (Vocal Separation & Mixing)
	// 尝试分离原音频的人声和背景音
	separationResult, sepErr := s.SeparateAudioSources(ctx, stepParam)
	
	videoWithTtsPath := filepath.Join(stepParam.TaskBasePath, types.SubtitleTaskVideoWithTtsFileName)
	
	if sepErr != nil {
		// 分离失败，回退到直接替换模式（无背景音乐）
		log.GetLogger().Warn("Audio separation failed, falling back to direct replacement",
			zap.String("taskId", stepParam.TaskId),
			zap.Error(sepErr))
		err = util.ReplaceAudioInVideo(stepParam.InputVideoPath, finalOutput, videoWithTtsPath)
	} else if separationResult.InstrumentalPath != "" {
		// 分离成功，将TTS与背景音混合
		log.GetLogger().Info("Audio separation successful, mixing TTS with instrumental",
			zap.String("taskId", stepParam.TaskId),
			zap.String("instrumentalPath", separationResult.InstrumentalPath))
		
		// 混合TTS和伴奏 (TTS音量1.0, 伴奏音量0.35)
		mixedAudioPath := filepath.Join(stepParam.TaskBasePath, "mixed_audio.aac")
		mixErr := util.MixAudioTracks(finalOutput, separationResult.InstrumentalPath, mixedAudioPath, 1.0, 0.35)
		
		if mixErr != nil {
			// 混音失败，回退到直接替换
			log.GetLogger().Warn("Audio mixing failed, falling back to direct replacement",
				zap.String("taskId", stepParam.TaskId),
				zap.Error(mixErr))
			err = util.ReplaceAudioInVideo(stepParam.InputVideoPath, finalOutput, videoWithTtsPath)
		} else {
			// 混音成功，使用混合后的音频
			err = util.ReplaceAudioInVideo(stepParam.InputVideoPath, mixedAudioPath, videoWithTtsPath)
		}
	} else {
		// 伴奏文件不存在，回退
		log.GetLogger().Warn("Instrumental track not found, falling back to direct replacement",
			zap.String("taskId", stepParam.TaskId))
		err = util.ReplaceAudioInVideo(stepParam.InputVideoPath, finalOutput, videoWithTtsPath)
	}
	
	if err != nil {
		log.GetLogger().Error("srtFileToSpeech ReplaceAudioInVideo error", zap.Any("stepParam", stepParam), zap.Error(err))
	}
	stepParam.VideoWithTtsFilePath = videoWithTtsPath
	// 更新字幕任务信息
	stepParam.TaskPtr.ProcessPct = 98
	log.GetLogger().Info("srtFileToSpeech success", zap.String("task id", stepParam.TaskId))
	return nil
}

func (s Service) processSubtitlesConcurrently(subtitles []types.SrtSentenceWithStrTime, voiceCode string, stepParam *types.SubtitleTaskStepParam) error {
	// 创建一个结果数组来存储每个字幕的处理结果
	type processingResult struct {
		index int
		err   error
	}

	maxConcurrency := 3 // 降低并发数以减少网络压力
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	resultCh := make(chan processingResult, len(subtitles))

	// 并发生成所有音频文件
	for i, sub := range subtitles {
		wg.Add(1)
		go func(index int, subtitle types.SrtSentenceWithStrTime) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			outputFile := filepath.Join(stepParam.TaskBasePath, fmt.Sprintf("subtitle_%d.wav", index+1))
			err := s.TtsClient.Text2Speech(subtitle.Text, voiceCode, outputFile)
			if err != nil {
				log.GetLogger().Error("processSubtitlesConcurrently Text2Speech error",
					zap.Any("index", index+1),
					zap.String("text", subtitle.Text),
					zap.Error(err))
				resultCh <- processingResult{index: index, err: fmt.Errorf("subtitle %d TTS error: %w", index+1, err)}
				return
			}

			// 成功处理
			resultCh <- processingResult{index: index, err: nil}
		}(i, sub)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(resultCh)

	// 收集所有结果并统计错误
	results := make([]processingResult, len(subtitles))
	errorCount := 0
	var firstError error

	for result := range resultCh {
		results[result.index] = result
		if result.err != nil {
			errorCount++
			if firstError == nil {
				firstError = result.err
			}
		}
	}

	// 如果有超过一半的字幕失败，则返回错误
	failureThreshold := len(subtitles) / 2
	if errorCount > failureThreshold {
		log.GetLogger().Error("processSubtitlesConcurrently: too many failures",
			zap.Int("total", len(subtitles)),
			zap.Int("errors", errorCount),
			zap.Int("threshold", failureThreshold))
		return fmt.Errorf("too many TTS failures: %d/%d failed, first error: %w", errorCount, len(subtitles), firstError)
	}

	// 验证成功的文件是否存在，对于失败的文件生成静音
	for i, result := range results {
		outputFile := filepath.Join(stepParam.TaskBasePath, fmt.Sprintf("subtitle_%d.wav", i+1))

		if result.err != nil {
			// 为失败的字幕生成静音文件
			log.GetLogger().Warn("生成静音文件替代失败的TTS",
				zap.Int("index", i+1),
				zap.String("file", outputFile))

			// 生成0.5秒的静音作为替代
			err := newGenerateSilence(outputFile, 0.5)
			if err != nil {
				log.GetLogger().Error("生成替代静音文件失败",
					zap.Int("index", i+1),
					zap.Error(err))
				return fmt.Errorf("failed to generate silence for subtitle %d: %w", i+1, err)
			}
		} else {
			// 验证成功生成的文件是否存在
			if _, err := os.Stat(outputFile); os.IsNotExist(err) {
				log.GetLogger().Error("processSubtitlesConcurrently output file not exist",
					zap.Any("index", i+1),
					zap.String("file", outputFile))
				return fmt.Errorf("subtitle %d output file not exist: %s", i+1, outputFile)
			}
		}
	}

	if errorCount > 0 {
		log.GetLogger().Warn("processSubtitlesConcurrently completed with some failures",
			zap.Int("total", len(subtitles)),
			zap.Int("errors", errorCount),
			zap.Int("success", len(subtitles)-errorCount))
	} else {
		log.GetLogger().Info("processSubtitlesConcurrently completed successfully", zap.Int("total", len(subtitles)))
	}

	return nil
}

func parseSRT(filePath string) ([]types.SrtSentenceWithStrTime, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("parseSRT read file error: %w", err)
	}

	var subtitles []types.SrtSentenceWithStrTime
	re := regexp.MustCompile(`(\d{2}:\d{2}:\d{2},\d{3}) --> (\d{2}:\d{2}:\d{2},\d{3})\s+(.+?)\n`)
	matches := re.FindAllStringSubmatch(string(data), -1)

	for _, match := range matches {
		subtitles = append(subtitles, types.SrtSentenceWithStrTime{
			Start: match[1],
			End:   match[2],
			Text:  strings.Replace(match[3], "\n", " ", -1), // 去除换行
		})
	}

	return subtitles, nil
}

func newGenerateSilence(outputAudio string, duration float64) error {
	// 生成 PCM 格式的静音文件
	cmd := exec.Command(storage.FfmpegPath, "-y", "-f", "lavfi", "-i", "anullsrc=channel_layout=mono:sample_rate=44100", "-t",
		fmt.Sprintf("%.3f", duration), "-ar", "44100", "-ac", "1", "-c:a", "pcm_s16le", outputAudio)
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("newGenerateSilence failed to generate PCM silence: %w", err)
	}

	return nil
}

// 调整音频时长，确保音频与字幕时长一致
func adjustAudioDuration(inputFile, outputFile, taskBasePath string, subtitleDuration float64) (float64, error) {
	// First, resample input to 44100Hz to avoid concatenation distortion
	resampledInput := filepath.Join(taskBasePath, "resampled_" + filepath.Base(inputFile))
	defer os.Remove(resampledInput)
	
	// Resample command (Force 44100Hz and Mono)
	cmdResample := exec.Command(storage.FfmpegPath, "-y", "-i", inputFile, "-ar", "44100", "-ac", "1", resampledInput)
	if err := cmdResample.Run(); err != nil {
		return 0, fmt.Errorf("resample input failed: %w", err)
	}

	// 获取音频时长
	audioDuration, err := util.GetAudioDuration(resampledInput)
	if err != nil {
		return 0, err
	}

	// 如果音频时长短于字幕时长，插入静音延长音频
	if audioDuration < subtitleDuration {
		// 计算需要插入的静音时长
		silenceDuration := subtitleDuration - audioDuration

		// 生成静音音频
		silenceFile := filepath.Join(taskBasePath, "silence_pad.wav")
		// Use newGenerateSilence which produces 44100Hz audio
		err := newGenerateSilence(silenceFile, silenceDuration)
		if err != nil {
			return 0, fmt.Errorf("error generating silence: %v", err)
		}
		defer os.Remove(silenceFile)

		// 拼接音频和静音
		concatFile := filepath.Join(taskBasePath, "concat_pad.txt")
		f, err := os.Create(concatFile)
		if err != nil {
			return 0, fmt.Errorf("adjustAudioDuration create concat file error: %w", err)
		}
		
		_, err = f.WriteString(fmt.Sprintf("file '%s'\nfile '%s'\n", filepath.Base(resampledInput), filepath.Base(silenceFile)))
		f.Close()
		defer os.Remove(concatFile)

		// -c copy is safe now because both are 44100Hz (resampledInput and silenceFile)
		cmd := exec.Command(storage.FfmpegPath, "-y", "-f", "concat", "-safe", "0", "-i", concatFile, "-c", "copy", outputFile)
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return 0, fmt.Errorf("adjustAudioDuration concat audio and silence error: %v", err)
		}

		finalDur, _ := util.GetAudioDuration(outputFile)
		return finalDur, nil
	}

	// 如果音频时长长于字幕时长，缩放音频的播放速率
	if audioDuration > subtitleDuration {
		// 计算播放速率
		speed := audioDuration / subtitleDuration
		
		// Uncommmented & Updated Limit Check
		// If speed is too high (speech too slow), we must maintain synch so we still speed up, 
		// but we might want to warn or limit if extreme.
		// Doubao/TTS is usually fast, so speed < 1.0 (slowing down) is rare.
		// Speed > 1.0 (speeding up) is common.
		// If speed > 1.5, it sounds bad. But better than desync?
		// Let's cap at 2.0 to prevent crash, but if > 2.0, we just chop? No, chopping loses text.
		// We process at max 2.0 and let it overlap (sync drift) is better than crash.
		// BUT system handles sync with gap logic now. If we return longer audio, next gap will be smaller.
		// So we strictly enforce duration unless it's impossible.
		
		if speed > 2.0 {
			log.GetLogger().Warn("adjustAudioDuration speed too high, clamping to 2.0", zap.Float64("speed", speed))
			speed = 2.0
		} else if speed < 0.5 {
			log.GetLogger().Warn("adjustAudioDuration speed too low, clamping to 0.5", zap.Float64("speed", speed))
			speed = 0.5
		}

		// 使用 atempo 滤镜调整音频播放速率 + 确保采样率44100
		cmd := exec.Command(storage.FfmpegPath, "-y", "-i", resampledInput, "-filter:a", fmt.Sprintf("atempo=%.2f", speed), "-ar", "44100", outputFile)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return 0, err
		}
		finalDur, _ := util.GetAudioDuration(outputFile)
		return finalDur, nil
	}

	// 如果音频时长和字幕时长相同，则直接复制文件 (resampled version)
	err = util.CopyFile(resampledInput, outputFile)
	if err != nil {
		return 0, err
	}
	finalDur, _ := util.GetAudioDuration(outputFile)
	return finalDur, nil
}

// 拼接音频文件
func concatenateAudioFiles(audioFiles []string, outputFile, taskBasePath string) error {
	// 创建一个临时文件保存音频文件列表
	listFile := filepath.Join(taskBasePath, "audio_list.txt")
	f, err := os.Create(listFile)
	if err != nil {
		return err
	}
	defer os.Remove(listFile)

	for _, file := range audioFiles {
		_, err := f.WriteString(fmt.Sprintf("file '%s'\n", filepath.Base(file)))
		if err != nil {
			return err
		}
	}
	f.Close()

	cmd := exec.Command(storage.FfmpegPath, "-y", "-f", "concat", "-safe", "0", "-i", listFile, "-c", "copy", outputFile)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}