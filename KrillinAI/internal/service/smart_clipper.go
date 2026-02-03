package service

import (
	"encoding/json"
	"fmt"
	"krillin-ai/config"
	"krillin-ai/internal/dto"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"krillin-ai/pkg/openai"
	"krillin-ai/pkg/util"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// In-memory cache for analysis results (Token -> CacheData)
var analysisCache sync.Map

// AnalyzeVideo download subtitles and ask AI to split video
func (s *Service) AnalyzeVideo(req dto.SmartClipperAnalyzeReq) (*dto.SmartClipperAnalyzeResData, error) {
	log.GetLogger().Info("SmartClipper: AnalyzeVideo", zap.String("url", req.Url))

	// 1. Create temporary directory for analysis
	tempDir := filepath.Join("data", "temp_analysis", util.GenerateRandStringWithUpperLowerNum(8))
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	// defer os.RemoveAll(tempDir) // Keep for debugging for now, or clean up later

	// 2. Download Subtitles and Video Info (Skip video download)
	// cmd: yt-dlp --skip-download --write-sub --write-auto-sub --sub-lang en,zh,ja --output "tempDir/%(title)s.%(ext)s" --print-json url
	cmdArgs := []string{
		"--skip-download",
		"--write-sub",
		"--write-auto-sub",
		"--sub-lang", "en,zh-Hans,zh-Hant,ja",
		"--output", filepath.Join(tempDir, "%(title)s.%(ext)s"),
		"--dump-json", // Output JSON info to stdout
		"--ignore-no-formats-error", // Ignore "Requested format is not available"
		req.Url,
	}
	
	if config.Conf.App.Proxy != "" {
		cmdArgs = append(cmdArgs, "--proxy", config.Conf.App.Proxy)
	}
	cmdArgs = append(cmdArgs, "--cookies", "/app/cookies.txt")

	cmd := exec.Command(storage.YtdlpPath, cmdArgs...)
	output, err := cmd.Output()
	if err != nil {
		log.GetLogger().Error("SmartClipper: yt-dlp failed", zap.Error(err))
		return nil, fmt.Errorf("failed to download subtitles info: %w", err)
	}

	// Parse Video Info
	var videoInfo struct {
		Title    string `json:"title"`
		Duration int    `json:"duration"`
		ID       string `json:"id"`
	}
	if err := json.Unmarshal(output, &videoInfo); err != nil {
		// Log error but proceed if likely just output noise? yt-dlp dump-json can be huge.
		// However, Output() captures stdout. yt-dlp outputs JSON only if --dump-json is used properly.
		// It might output multiple lines if playlist. Assuming single video.
		// Let's try to unmarshal the first line if multiple.
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if len(line) > 10 && json.Unmarshal([]byte(line), &videoInfo) == nil {
				break
			}
		}
	}
	
	// 3. Find Subtitle File (.vtt)
	var subFile string
	entries, _ := os.ReadDir(tempDir)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".vtt") {
			subFile = filepath.Join(tempDir, entry.Name())
			break
		}
	}
	if subFile == "" {
		return nil, fmt.Errorf("no subtitles found for video (auto-generated or manual)")
	}

	// 4. Read and Parse Subtitles to Text
	subContent, err := os.ReadFile(subFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read subtitle file: %w", err)
	}
	// Simple VTT text extraction (Removing timestamps and metadata lines)
	// Or just pass the whole thing if Kimi supports it? 
	// To save tokens, let's do a simple cleanup.
	cleanText := cleanVttContent(string(subContent))

	// 5. Call Kimi (LLM) for Analysis
	basePrompt := config.Conf.SmartClipper.Prompt
	if basePrompt == "" {
		basePrompt = types.SmartClipperPrompt
	}
	prompt := fmt.Sprintf(basePrompt, config.Conf.SmartClipper.MinClipDuration, config.Conf.SmartClipper.MaxClipDuration, cleanText)

	// Determine Config (Fallback to global LLM config if specific one is empty)
	scBaseUrl := config.Conf.SmartClipper.BaseUrl
	scApiKey := config.Conf.SmartClipper.ApiKey
	
	if scApiKey == "" {
		// Fallback to main LLM config
		// Use Transcribe.Openai if provider is openai? Or Llm?
		// User mentioned "Kimi API Key already provided" for translation. 
		// Usually translation uses Llm config or Transcribe config?
		// Looking at config.go: Conf.Llm is OpenaiCompatibleConfig. 
		// Let's assume Conf.Llm is the one.
		scBaseUrl = config.Conf.Llm.BaseUrl
		scApiKey = config.Conf.Llm.ApiKey
		log.GetLogger().Info("SmartClipper: Using global LLM config", zap.String("baseUrl", scBaseUrl))
	} else {
		log.GetLogger().Info("SmartClipper: Using specific SmartClipper config", zap.String("baseUrl", scBaseUrl))
	}

	// Create specific client for Kimi
	kimiClient := openai.NewClient(
		scBaseUrl,
		scApiKey,
		config.Conf.App.Proxy,
	)

	// Call ChatCompletion (Non-streaming for simplicity in backend logic, but client only has streaming implemented in openai.go? 
	// openai.go: ChatCompletion returns string but uses stream internally. That's fine.)
	llmResp, err := kimiClient.ChatCompletion(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// 6. Parse LLM JSON Response
	// The LLM might wrap JSON in ```json code blocks. Clean it.
	jsonStr := util.ExtractJsonFromText(llmResp)
	var clips []dto.TopicClip
	if err := json.Unmarshal([]byte(jsonStr), &clips); err != nil {
		log.GetLogger().Error("SmartClipper: failed to parse LLM JSON", zap.String("response", llmResp), zap.Error(err))
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// 7. Generate Token and Cache
	token := uuid.New().String()
	cacheData := types.SmartClipperCacheData{
		VideoId:      videoInfo.ID,
		VideoTitle:   videoInfo.Title,
		VideoPath:    "", // Not downloaded yet
		SubtitlePath: subFile,
		Clips:        convertDtoClipsToCache(clips),
		CreatedAt:    time.Now(),
	}
	analysisCache.Store(token, cacheData)

	return &dto.SmartClipperAnalyzeResData{
		VideoTitle: videoInfo.Title,
		Duration:   fmt.Sprintf("%d", videoInfo.Duration),
		Clips:      clips,
		Token:      token,
	}, nil
}

// SubmitClips processes selected clips
func (s *Service) SubmitClips(req dto.SmartClipperSubmitReq) (*dto.SmartClipperSubmitResData, error) {
	// 1. Retrieve Cache
	val, ok := analysisCache.Load(req.Token)
	if !ok {
		return nil, fmt.Errorf("session expired or invalid token")
	}
	data := val.(types.SmartClipperCacheData)

	// 2. Download Full Video (if not present) 
	// Currently data.VideoPath is empty. We need to download it.
	// We create a "master" task folder for this video source
	masterTaskId := fmt.Sprintf("master_%s", data.VideoId)
	masterTaskDir := filepath.Join("tasks", masterTaskId)
	os.MkdirAll(masterTaskDir, os.ModePerm)
	
	masterVideoPath := filepath.Join(masterTaskDir, "master.mp4")
	if _, err := os.Stat(masterVideoPath); os.IsNotExist(err) {
		// Download Logic
		cmdArgs := []string{
			"-f", "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
			"-o", masterVideoPath,
			"https://www.youtube.com/watch?v=" + data.VideoId,
		}
		if config.Conf.App.Proxy != "" {
			cmdArgs = append(cmdArgs, "--proxy", config.Conf.App.Proxy)
		}
		cmdArgs = append(cmdArgs, "--cookies", "/app/cookies.txt")
		
		log.GetLogger().Info("SmartClipper: Downloading master video", zap.String("id", data.VideoId))
		cmd := exec.Command(storage.YtdlpPath, cmdArgs...)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.GetLogger().Error("SmartClipper: video download failed", zap.String("output", string(out)))
			return nil, fmt.Errorf("failed to download master video: %w", err)
		}
	}

	// 3. Process each clip
	var taskIds []string
	
	for _, clipId := range req.SelectedClipIds {
		// Find clip info
		var targetClip *types.ClipInfo
		for _, c := range data.Clips {
			if c.Id == clipId {
				targetClip = &c
				break
			}
		}
		if targetClip == nil {
			continue
		}

		// 3.1 Split Video using ffmpeg
		// Task ID for child task
		childTaskId := fmt.Sprintf("clip_%s_%d", data.VideoId, clipId)
		childTaskDir := filepath.Join("tasks", childTaskId)
		os.MkdirAll(childTaskDir, os.ModePerm)
		
		clipVideoPath := filepath.Join(childTaskDir, "origin_video.mp4") // naming convention for StartSubtitleTask input?
		// NOTE: StartSubtitleTask usually expects a URL or local file path.
		
		// ffmpeg -ss START -to END -i master -c copy output
		// -c copy is fast but might inaccurate at keyframes. If re-encoding needed, remove "-c copy"
		// StartSubtitleTask will process it again anyway, so maybe copy is fine for source, 
		// BUT StartSubtitleTask flow downloads the video itself if URL provided.
		// If we provide "local:/path/to/video", linkToFile supports it (I saw it in link2file.go: line 27)
		
		cmd := exec.Command(storage.FfmpegPath, 
			"-y", "-ss", targetClip.Start, "-to", targetClip.End, 
			"-i", masterVideoPath, 
			"-c", "copy", // Try stream copy first
			"-avoid_negative_ts", "1",
			clipVideoPath,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.GetLogger().Error("SmartClipper: split failed", zap.Int("clipId", clipId), zap.String("out", string(out)))
			// Fallback to re-encode if copy fails or is erratic?
			// For now error out
			continue
		}

		// 3.2 Create Subtitle Task
		// Clone request params
		taskReq := req.TaskParams
		taskReq.Url = "local:" + clipVideoPath // Magic prefix for local file
		taskReq.ReuseTaskId = childTaskId // Use predicted ID
		
		// Call StartSubtitleTask
		_, err := s.StartSubtitleTask(taskReq)
		if err != nil {
			log.GetLogger().Error("SmartClipper: failed to start task", zap.String("childId", childTaskId), zap.Error(err))
		} else {
			taskIds = append(taskIds, childTaskId)
		}
	}

	return &dto.SmartClipperSubmitResData{
		TaskIds: taskIds,
	}, nil
}

// Helpers

func cleanVttContent(content string) string {
	lines := strings.Split(content, "\n")
	var sb strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "WEBVTT" || strings.Contains(line, "-->") || line == "" {
			continue
		}
		// Skip purely numeric lines (sequence numbers) - simple heuristic
		// or just append everything, LLM can handle it.
		// Let's rely on Kimi's capability. Just filter header mainly.
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

func convertDtoClipsToCache(dtoClips []dto.TopicClip) []types.ClipInfo {
	var res []types.ClipInfo
	for _, c := range dtoClips {
		res = append(res, types.ClipInfo{
			Id: c.Id, Title: c.Title, Summary: c.Summary,
			Start: c.Start, End: c.End, Duration: c.Duration,
		})
	}
	return res
}
