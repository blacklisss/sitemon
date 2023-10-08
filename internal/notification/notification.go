package notification

//go:generate go run github.com/vektra/mockery/v2@v2.35.2 --name=Notificator
type Notificator interface {
	SendMessage(message string) error
}
