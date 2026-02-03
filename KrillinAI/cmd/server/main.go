package main

import (
	"go.uber.org/zap"
	"krillin-ai/config"
	"krillin-ai/internal/deps"
	"krillin-ai/internal/server"
	"krillin-ai/internal/storage"
	"krillin-ai/log"
	"os"
)

func main() {
	log.InitLogger()
	defer log.GetLogger().Sync()

	var err error
	if !config.LoadConfig() {
		return
	}

	if err = config.CheckConfig(); err != nil {
		log.GetLogger().Error("加载配置失败", zap.Error(err))
		return
	}

	// Initialize Database
	storage.InitDB()

	// Mark any stale "running" tasks as "failed" (zombie cleanup)
	if count, err := storage.MarkStaleTasks(); err != nil {
		log.GetLogger().Warn("Failed to mark stale tasks", zap.Error(err))
	} else if count > 0 {
		log.GetLogger().Info("Marked stale tasks as failed", zap.Int64("count", count))
	}

	if err = deps.CheckDependency(); err != nil {
		log.GetLogger().Error("依赖环境准备失败", zap.Error(err))
		return
	}
	if err = server.StartBackend(); err != nil {
		log.GetLogger().Error("后端服务启动失败", zap.Error(err))
		os.Exit(1)
	}
}
