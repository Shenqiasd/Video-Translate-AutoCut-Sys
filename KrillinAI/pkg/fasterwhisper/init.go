package fasterwhisper

import "os"

type FastwhisperProcessor struct {
	WorkDir string // 生成中间文件的目录
	Model   string
}

func NewFastwhisperProcessor(model string) *FastwhisperProcessor {
	// 设置 HuggingFace 镜像，加速国内下载
	_ = os.Setenv("HF_ENDPOINT", "https://hf-mirror.com")
	return &FastwhisperProcessor{
		Model: model,
	}
}
