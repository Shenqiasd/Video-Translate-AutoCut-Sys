package storage

import (
	"krillin-ai/internal/appdirs"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var appDirsResolver = appdirs.Resolve

func InitDB() {
	dbPath, err := resolveDBPath()
	if err != nil {
		log.GetLogger().Fatal("failed to resolve database path", zap.Error(err))
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.GetLogger().Fatal("failed to create database directory", zap.String("dir", dir), zap.Error(err))
	}

	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.GetLogger().Fatal("failed to connect database", zap.Error(err))
	}

	// Auto Migrate the schema
	err = DB.AutoMigrate(&types.SubtitleTask{}, &types.SubtitleInfo{})
	if err != nil {
		log.GetLogger().Fatal("failed to migrate database", zap.Error(err))
	}

	log.GetLogger().Info("Database initialized successfully", zap.String("path", dbPath))
}

func resolveDBPath() (string, error) {
	dirs, err := appDirsResolver()
	if err != nil {
		return "", err
	}
	return appdirs.DBPathFor(dirs), nil
}
