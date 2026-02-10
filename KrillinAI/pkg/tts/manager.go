package tts

import (
	"krillin-ai/config"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"krillin-ai/pkg/aliyun"
	"krillin-ai/pkg/doubao"
	"krillin-ai/pkg/localtts"
	"krillin-ai/pkg/minimax"
	"krillin-ai/pkg/openai"
	"strings"

	"go.uber.org/zap"
)

// CompositeTtsClient manages multiple TTS providers and routes requests
type CompositeTtsClient struct {
	Doubao  *doubao.DoubaoClient
	EdgeTTS *localtts.EdgeTtsClient
	OpenAI  *openai.Client
	Aliyun  *aliyun.TtsClient
	MiniMax *minimax.MiniMaxClient
	Default types.Ttser
}

func NewCompositeTtsClient() *CompositeTtsClient {
	c := &CompositeTtsClient{}

	// Initialize Clients based on config availability
	// We verify config presence lightly, but actual errors happen at call time if creds are bad

	// EdgeTTS (Always available as fallback)
	c.EdgeTTS = localtts.NewEdgeTtsClient()

	// Doubao
	if config.Conf.Tts.Doubao.AppId != "" {
		c.Doubao = doubao.NewDoubaoClient(
			config.Conf.Tts.Doubao.AppId,
			config.Conf.Tts.Doubao.AccessToken,
			config.Conf.Tts.Doubao.Cluster,
		)
	}

	// OpenAI
	if config.Conf.Tts.Openai.ApiKey != "" {
		c.OpenAI = openai.NewClient(config.Conf.Tts.Openai.BaseUrl, config.Conf.Tts.Openai.ApiKey, config.Conf.App.Proxy)
	}

	// MiniMax
	if config.Conf.Tts.Minimax.ApiKey != "" {
		c.MiniMax = minimax.NewMiniMaxClient(config.Conf.Tts.Minimax.ApiKey, config.Conf.Tts.Minimax.GroupId, config.Conf.Tts.Minimax.Model)
	}

	// Set Default based on Config
	switch config.Conf.Tts.Provider {
	case "aliyun":
		c.Aliyun = aliyun.NewTtsClient(config.Conf.Tts.Aliyun.Speech.AccessKeyId, config.Conf.Tts.Aliyun.Speech.AccessKeySecret, config.Conf.Tts.Aliyun.Speech.AppKey)

	case "doubao":
		c.Default = c.Doubao
	case "openai":
		c.Default = c.OpenAI
	case "minimax":
		c.Default = c.MiniMax
	case "edge-tts":
		c.Default = c.EdgeTTS
	default:
		c.Default = c.EdgeTTS
	}

	// Safety fallback
	if c.Default == nil {
		c.Default = c.EdgeTTS
	}

	return c
}

func (c *CompositeTtsClient) Text2Speech(text, voice, outputFile string) error {
	// Routing Logic

	// 1. Check for Doubao specific patterns
	if strings.Contains(voice, "bigtts") || strings.Contains(voice, "mars") || strings.Contains(voice, "moon") || strings.Contains(voice, "volcano") {
		if c.Doubao != nil {
			log.GetLogger().Info("Routing to Doubao TTS", zap.String("voice", voice))
			return c.Doubao.Text2Speech(text, voice, outputFile)
		}
	}

	// 2. Check for Edge TTS specific patterns (zh-CN-...)
	// Most Edge TTS voices follow "zh-CN-XiaoxiaoNeural" format
	if (strings.HasPrefix(voice, "zh-CN-") || strings.HasPrefix(voice, "en-US-")) && strings.Contains(voice, "Neural") {
		log.GetLogger().Info("Routing to Edge TTS", zap.String("voice", voice))
		return c.EdgeTTS.Text2Speech(text, voice, outputFile)
	}

	// 3. MiniMax check (usually short IDs or specific names, tough to distinguish without list)
	// Assuming user config selects MiniMax if they want it main, generally.

	// 4. Volc speaker-id (voice clone) requires ICL cluster
	if IsVolcSpeakerID(voice) {
		if c.Doubao != nil {
			if c.Doubao.Cluster != "volcano_icl" && c.Doubao.Cluster != "volcano_icl_concurr" {
				c.Doubao.Cluster = "volcano_icl"
			}
			log.GetLogger().Info("Routing to Doubao TTS (ICL clone)", zap.String("voice", voice), zap.String("cluster", c.Doubao.Cluster))
			return c.Doubao.Text2Speech(text, voice, outputFile)
		}
	}

	// 5. Fallback to Default
	log.GetLogger().Info("Routing to Default TTS", zap.String("voice", voice), zap.Any("default_provider", config.Conf.Tts.Provider))
	return c.Default.Text2Speech(text, voice, outputFile)
}
