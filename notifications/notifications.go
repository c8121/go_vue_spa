package notifications

import "fmt"

var (
	AppNotifications = NewNotifications()
)

type Notifications struct {
	MaxNotifications int
	Messages         []string
}

func NewNotifications() *Notifications {
	return &Notifications{
		MaxNotifications: 25,
		Messages:         make([]string, 0),
	}
}

func (n *Notifications) AddNotification(notification string) {
	n.Messages = append(n.Messages, notification)
	if len(n.Messages) > n.MaxNotifications {
		n.Messages = n.Messages[1:]
	}
}

func (n *Notifications) PrintNotifications() {
	for _, notification := range n.Messages {
		fmt.Println(notification)
	}
}
