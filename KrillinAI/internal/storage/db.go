package storage

import (
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

func InitDB() {
	var err error
	dbPath := "data/krillin.db"
	
	// Ensure data directory exists
	dir := filepath.Dir(dbPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
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
