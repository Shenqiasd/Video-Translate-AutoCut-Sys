package openai

import (
	"github.com/sashabaranov/go-openai"
	"krillin-ai/config"
	"net/http"
)

type Client struct {
	client *openai.Client
}

func NewClient(baseUrl, apiKey, proxyAddr string) *Client {
	cfg := openai.DefaultConfig(apiKey)
	if baseUrl != "" {
		cfg.BaseURL = baseUrl
	}

	// 总是配置自定义 HTTP Client 以设置超时
	transport := &http.Transport{}
	if proxyAddr != "" {
		transport.Proxy = http.ProxyURL(config.Conf.App.ParsedProxy)
	}

	cfg.HTTPClient = &http.Client{
		Transport: transport,
		// 不设置超时，允许长时间运行的翻译请求（Thinking 模型可能需要 30+ 分钟）
	}

	client := openai.NewClientWithConfig(cfg)
	return &Client{client: client}
}
