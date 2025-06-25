package logger

var Production string

type Logger interface {
	Debug(template string, args ...any)
	Info(template string, args ...any)
	Warn(template string, args ...any)
	Error(err error)
	Panic(err error)
	Fatal(err error)
}

func isProduction() bool {
	return Production == "prod"
}
