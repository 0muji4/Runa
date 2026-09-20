// Package push defines the provider-neutral push-notification seam; the apns
// and fcm subpackages implement it, mempush fakes it.
package push

import (
	"context"
	"errors"
	"time"
)

// Platform values match the devices.platform column.
const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
)

var (
	// ErrInvalidToken means the provider rejected the device token for good
	// (unregistered, malformed, wrong app); the caller should stop using it.
	ErrInvalidToken = errors.New("push: invalid device token")
	// ErrTransient means the provider could not accept the message right now
	// (rate limit, 5xx, network); the caller may retry later.
	ErrTransient = errors.New("push: transient provider failure")
)

// Notification is one message for one device; Data reaches the client as string key/values.
type Notification struct {
	Title string
	Body  string
	Data  map[string]string
	// CollapseID makes a redelivery replace the previous tray entry instead of stacking.
	CollapseID string
	// Expiry is when the provider should stop trying to deliver to an offline device.
	Expiry time.Time
}

// Result carries the provider's message id for log correlation.
type Result struct {
	ProviderMessageID string
}

// Sender delivers to one platform; errors wrap ErrInvalidToken or ErrTransient
// when the caller should react.
type Sender interface {
	Send(ctx context.Context, token string, n Notification) (Result, error)
}
