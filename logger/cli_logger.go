package logger

import (
	"fmt"
	"os"
)

type CliLogger struct{}

func NewCliLogger() Logger {
	if isProduction() {
		return &CliLogger{}
	}
	return NewServiceLogger("")
}

func (cl *CliLogger) Debug(template string, args ...any) {}

func (cl *CliLogger) Info(template string, args ...any) {
	fmt.Printf(template+"\n", args...)
}

func (cl *CliLogger) Warn(template string, args ...any) {
	fmt.Fprintf(os.Stderr, template+"\n", args...)
}

func (cl *CliLogger) Error(err error) {
	fmt.Fprint(os.Stderr, err, "\n")
}

func (cl *CliLogger) Panic(err error) {
	panic(err)
}

func (cl *CliLogger) Fatal(err error) {
	fmt.Fprint(os.Stderr, err, "\n")
	os.Exit(1)
}
