package logger

import (
	"io"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

type log struct{}

var logInfo = logrus.New()
var logError = logrus.New()

var Logs = &log{}

func init() {

	writerInfo, _ := rotatelogs.New(
		"log/info_%Y%m%d.log",
		rotatelogs.WithMaxAge(30*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	logInfo.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logInfo.SetOutput(io.MultiWriter(writerInfo))

	writerError, _ := rotatelogs.New(
		"log/error_%Y%m%d.log",
		rotatelogs.WithMaxAge(30*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	logError.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logError.SetOutput(io.MultiWriter(writerError))
}

// Info log
func (log *log) Info(args ...interface{}) {
	logInfo.Info(args...)
}

// Infof log
func (log *log) Infof(format string, args ...interface{}) {
	logInfo.Infof(format, args...)
}

// Debug log
func (log *log) Debug(args ...interface{}) {
	logInfo.Debug(args...)
}

// Println log
func (log *log) Println(args ...interface{}) {
	logInfo.Println(args...)
}

// Error log
func (log *log) Error(args ...interface{}) {
	logError.Error(args...)
}

// Errorf log
func (log *log) Errorf(format string, args ...interface{}) {
	logError.Errorf(format, args...)
}

// Panic log
func (log *log) Panic(args ...interface{}) {
	logError.Panic(args...)
}
