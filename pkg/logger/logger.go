package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"time"
)

var logger *zap.Logger
var logLevel zap.AtomicLevel
var fileEncoder zapcore.Encoder
var consoleEncoder zapcore.Encoder
var writer zapcore.WriteSyncer

func Init() {
	customTimeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}

	config := zap.NewProductionEncoderConfig()
	config.EncodeLevel = zapcore.LowercaseLevelEncoder
	config.EncodeTime = customTimeEncoder

	fileEncoder = zapcore.NewJSONEncoder(config)
	consoleEncoder = zapcore.NewConsoleEncoder(config)

	writer = zapcore.AddSync(&lumberjack.Logger{
		Filename:   "logs/stream-recorder.log",
		MaxSize:    64,
		MaxBackups: 3,
		MaxAge:     30,
		Compress:   true,
	})

	logLevel = zap.NewAtomicLevel()
	logLevel.SetLevel(zapcore.InfoLevel)

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, logLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), logLevel),
	)
	logger = zap.New(core, zap.AddStacktrace(zapcore.FatalLevel))
}

func SetLogLevel(level string) {
	switch level {
	case "debug":
		logLevel.SetLevel(zapcore.DebugLevel)
	case "warn":
		logLevel.SetLevel(zapcore.WarnLevel)
	case "error":
		logLevel.SetLevel(zapcore.ErrorLevel)
	case "fatal":
		logLevel.SetLevel(zapcore.FatalLevel)
	case "info":
		logLevel.SetLevel(zapcore.InfoLevel)
	default:
		logLevel.SetLevel(zapcore.InfoLevel)
	}
}

func Debug(message string, fields ...zap.Field) {
	logger.Debug(message, fields...)
}

func Debugf(message string, username, platform string, fields ...zap.Field) {
	logger.Debug(fmt.Sprintf("[%s/%s] ", platform, username)+message, fields...)
}

func Info(message string, fields ...zap.Field) {
	logger.Info(message, fields...)
}

func Infof(message string, username, platform string, fields ...zap.Field) {
	logger.Info(fmt.Sprintf("[%s/%s] ", platform, username)+message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	logger.Warn(message, fields...)
}

func Warnf(message string, username, platform string, fields ...zap.Field) {
	logger.Warn(fmt.Sprintf("[%s/%s] ", platform, username)+message, fields...)
}

func Error(message string, fields ...zap.Field) {
	logger.Error(message, fields...)
}

func Errorf(message string, username, platform string, fields ...zap.Field) {
	logger.Error(fmt.Sprintf("[%s/%s] ", platform, username)+message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
	logger.Fatal(message, fields...)
}

func Fatalf(message string, username, platform string, fields ...zap.Field) {
	logger.Fatal(fmt.Sprintf("[%s/%s] ", platform, username)+message, fields...)
}
