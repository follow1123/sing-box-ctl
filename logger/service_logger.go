package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ServiceLogger struct {
	log *zap.SugaredLogger
}

func NewServiceLogger(logFile string) *ServiceLogger {
	zapConf := zap.NewDevelopmentConfig()
	zapConf.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	// 生产环境配置
	if isProduction() {
		zapConf.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		zapConf.Development = false
		if logFile != "" {
			zapConf.OutputPaths = []string{logFile}
		}
	}

	zapLogger, err := zapConf.Build()
	if err != nil {
		fmt.Fprint(os.Stderr, fmt.Errorf("init zap logger error: \n\t%w", err))
		os.Exit(1)
	}
	return &ServiceLogger{
		log: zapLogger.Sugar().WithOptions(zap.AddCallerSkip(1)),
	}
}

func (sl *ServiceLogger) Debug(template string, args ...any) {
	sl.log.Debugf(template, args...)
}

func (sl *ServiceLogger) Info(template string, args ...any) {
	sl.log.Infof(template, args...)
}

func (sl *ServiceLogger) Warn(template string, args ...any) {
	sl.log.Warnf(template, args...)
}

func (sl *ServiceLogger) Error(err error) {
	sl.log.Error(err)
}

func (sl *ServiceLogger) Panic(err error) {
	sl.log.Panic(err)
}

func (sl *ServiceLogger) Fatal(err error) {
	sl.log.Fatal(err)
}

func (sl *ServiceLogger) Sync() error {
	return sl.log.Sync()
}
