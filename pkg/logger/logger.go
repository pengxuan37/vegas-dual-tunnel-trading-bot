package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Logger 日志接口
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

// logrusLogger logrus实现
type logrusLogger struct {
	*logrus.Logger
}

// NewLogger 创建新的日志实例
func NewLogger() Logger {
	logger := logrus.New()
	
	// 设置输出格式
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	
	// 设置输出到标准输出
	logger.SetOutput(os.Stdout)
	
	// 设置日志级别
	logger.SetLevel(logrus.InfoLevel)
	
	return &logrusLogger{Logger: logger}
}

// NewLoggerWithLevel 创建指定级别的日志实例
func NewLoggerWithLevel(level string) Logger {
	logger := logrus.New()
	
	// 设置输出格式
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	
	// 设置输出到标准输出
	logger.SetOutput(os.Stdout)
	
	// 解析并设置日志级别
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)
	
	return &logrusLogger{Logger: logger}
}