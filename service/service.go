package service

type Service interface {
	Start() error
	Stop() error
	Restart() error
	CheckConfig(data []byte) error
	IsRunning() bool
}
