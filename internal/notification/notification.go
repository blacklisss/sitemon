package notification

type Notificator interface {
	SendMessage(message string) error
}
