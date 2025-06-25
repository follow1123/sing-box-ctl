package logger

import "testing"

type TestLogger struct {
	t *testing.T
}

func NewTestLogger(t *testing.T) *TestLogger {
	return &TestLogger{t: t}
}

func (tl *TestLogger) Debug(template string, args ...any) {
	tl.t.Logf(template, args...)
}

func (tl *TestLogger) Info(template string, args ...any) {
	tl.t.Logf(template, args...)
}

func (tl *TestLogger) Warn(template string, args ...any) {
	tl.t.Logf(template, args...)
}

func (tl *TestLogger) Error(err error) {
	tl.t.Log(err)
}

func (tl *TestLogger) Panic(err error) {
	tl.t.Log(err)
}

func (tl *TestLogger) Fatal(err error) {
	tl.t.Log(err)
}
