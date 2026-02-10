package service

import (
	"krillin-ai/config"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"krillin-ai/pkg/aliyun"
	"krillin-ai/pkg/doubao"
	"krillin-ai/pkg/fasterwhisper"
	"krillin-ai/pkg/openai"
	"krillin-ai/pkg/tts"
	"krillin-ai/pkg/whisper"
	"krillin-ai/pkg/whispercpp"
	"krillin-ai/pkg/whisperkit"

	"go.uber.org/zap"
)

type Service struct {
	Transcriber      types.Transcriber
	ChatCompleter    types.ChatCompleter
	TtsClient        types.Ttser
	OssClient        *aliyun.OssClient
	VoiceCloneClient *doubao.VoiceCloneClient
}

func NewService() *Service {
	var transcriber types.Transcriber
	var chatCompleter types.ChatCompleter
	var ttsClient types.Ttser

	switch config.Conf.Transcribe.Provider {
	case "openai":
		transcriber = whisper.NewClient(config.Conf.Transcribe.Openai.BaseUrl, config.Conf.Transcribe.Openai.ApiKey, config.Conf.App.Proxy)
	case "fasterwhisper":
		transcriber = fasterwhisper.NewFastwhisperProcessor(config.Conf.Transcribe.Fasterwhisper.Model)
	case "whispercpp":
		transcriber = whispercpp.NewWhispercppProcessor(config.Conf.Transcribe.Whispercpp.Model)
	case "whisperkit":
		transcriber = whisperkit.NewWhisperKitProcessor(config.Conf.Transcribe.Whisperkit.Model)
	case "aliyun":
		cc, err := aliyun.NewAsrClient(config.Conf.Transcribe.Aliyun.Speech.AccessKeyId, config.Conf.Transcribe.Aliyun.Speech.AccessKeySecret, config.Conf.Transcribe.Aliyun.Speech.AppKey, true)
		if err != nil {
			log.GetLogger().Error("创建阿里云语音识别客户端失败： ", zap.Error(err))
			return nil
		}
		transcriber = cc
	}
	log.GetLogger().Info("当前选择的转录源： ", zap.String("transcriber", config.Conf.Transcribe.Provider))

	chatCompleter = openai.NewClient(config.Conf.Llm.BaseUrl, config.Conf.Llm.ApiKey, config.Conf.App.Proxy)

	// Use Composite TTS Client to support multiple providers dynamically
	ttsClient = tts.NewCompositeTtsClient()

	return &Service{
		Transcriber:      transcriber,
		ChatCompleter:    chatCompleter,
		TtsClient:        ttsClient,
		OssClient:        aliyun.NewOssClient(config.Conf.Transcribe.Aliyun.Oss.AccessKeyId, config.Conf.Transcribe.Aliyun.Oss.AccessKeySecret, config.Conf.Transcribe.Aliyun.Oss.Bucket),
		VoiceCloneClient: doubao.NewVoiceCloneClient(config.Conf.Tts.VoiceCloneVolc.AppId, config.Conf.Tts.VoiceCloneVolc.AccessToken, config.Conf.Tts.VoiceCloneVolc.ResourceId),
	}
}
