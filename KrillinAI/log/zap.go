package log

import (
	"krillin-ai/internal/appdirs"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var Logger *zap.Logger

const logFileName = "app.log"

var appDirsResolver = appdirs.Resolve

func InitLogger() {
	logDir, err := ResolveLogDir()
	if err != nil {
		panic("无法解析日志目录: " + err.Error())
	}

	if err = os.MkdirAll(logDir, 0o755); err != nil {
		panic("无法创建日志目录: " + err.Error())
	}

	logFilePath := filepath.Join(logDir, logFileName)
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		panic("无法打开日志文件: " + err.Error())
	}

	fileSyncer := zapcore.AddSync(file)
	consoleSyncer := zapcore.AddSync(os.Stdout)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), fileSyncer, zap.DebugLevel),      // 写入文件（JSON 格式）
		zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), consoleSyncer, zap.InfoLevel), // 输出到终端
	)

	Logger = zap.New(core, zap.AddCaller())
}

func ResolveLogDir() (string, error) {
	dirs, err := appDirsResolver()
	if err != nil {
		return "", err
	}

	logDir := strings.TrimSpace(dirs.LogDir)
	if logDir == "" {
		return ".", nil
	}

	return logDir, nil
}

func ResolveLogFilePath() (string, error) {
	logDir, err := ResolveLogDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(logDir, logFileName), nil
}

func GetLogger() *zap.Logger {
	return Logger
}
