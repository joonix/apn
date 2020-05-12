package proto

type NotificationPayload interface {
	NotificationPayload() Notification
}

// Notification contains the required fields for the apple JSON API.
// It may be embedded in other structs that provide custom fields which also will
// be delivered by the remote notification service.
// https://developer.apple.com/library/prerelease/content/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/PayloadKeyReference.html#//apple_ref/doc/uid/TP40008194-CH17-SW1
type Notification struct {
	APS NotificationAPS `json:"aps"`
}

func (n Notification) NotificationPayload() Notification {
	return n
}

type NotificationAPS struct {
	Alert NotificationAlert `json:"alert,omitempty"`
	Badge int               `json:"badge,omitempty"`
	Sound string            `json:"sound,omitempty"`
	// ContentAvailable with a value of 1 will configure a background update notification.
	ContentAvailable int    `json:"content-available,omitempty"`
	Category         string `json:"category,omitempty"`
}

type NotificationAlert struct {
	Title        string   `json:"title,omitempty"`
	Body         string   `json:"body,omitempty"`
	ActionLocKey string   `json:"action-loc-key,omitempty"`
	LocKey       string   `json:"loc-key,omitempty"`
	LocArgs      []string `json:"loc-args,omitempty"`
}
