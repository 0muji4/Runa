package main

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/config"
	"github.com/0muji4/Runa/apps/go/internal/push"
	"github.com/0muji4/Runa/apps/go/internal/push/apns"
	"github.com/0muji4/Runa/apps/go/internal/push/fcm"
	"github.com/0muji4/Runa/apps/go/internal/schedule"
	"github.com/0muji4/Runa/apps/go/internal/schedule/cloudtasks"
)

// providerTimeout bounds one APNs / FCM / Cloud Tasks request.
const providerTimeout = 10 * time.Second

// newPushSenders builds one sender per configured platform; an unconfigured
// platform is absent from the map and its devices get no reminder.
func newPushSenders(cfg config.Config, logger *slog.Logger) map[string]push.Sender {
	senders := map[string]push.Sender{}

	if cfg.APNSPrivateKey == "" {
		logger.Info("push: APNs not configured; iOS reminders disabled")
	} else if sender, err := newAPNSSender(cfg); err != nil {
		logger.Warn("push: APNs disabled: init failed", slog.Any("error", err))
	} else {
		senders[push.PlatformIOS] = sender
	}

	if cfg.GCPServiceAccountJSON == "" {
		logger.Info("push: FCM not configured; Android reminders disabled")
	} else if sender, err := newFCMSender(cfg); err != nil {
		logger.Warn("push: FCM disabled: init failed", slog.Any("error", err))
	} else {
		senders[push.PlatformAndroid] = sender
	}
	return senders
}

func newAPNSSender(cfg config.Config) (push.Sender, error) {
	pem, err := base64.StdEncoding.DecodeString(cfg.APNSPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("APNS_PRIVATE_KEY is not base64: %w", err)
	}
	host, err := apns.HostFor(cfg.APNSEnvironment)
	if err != nil {
		return nil, err
	}
	return apns.NewClient(apns.Config{
		TeamID:        cfg.APNSTeamID,
		KeyID:         cfg.APNSKeyID,
		PrivateKeyPEM: pem,
		BundleID:      cfg.APNSBundleID,
		Host:          host,
	}, &http.Client{Timeout: providerTimeout})
}

func newFCMSender(cfg config.Config) (push.Sender, error) {
	sa, err := base64.StdEncoding.DecodeString(cfg.GCPServiceAccountJSON)
	if err != nil {
		return nil, fmt.Errorf("GCP_SERVICE_ACCOUNT_JSON is not base64: %w", err)
	}
	return fcm.NewClient(fcm.Config{
		ProjectID:          cfg.GCPProjectID,
		ServiceAccountJSON: sa,
	}, &http.Client{Timeout: providerTimeout})
}

// newScheduler builds the Cloud Tasks scheduler, or nil when any of its
// settings is missing (registrations then schedule nothing).
func newScheduler(cfg config.Config, logger *slog.Logger) schedule.Scheduler {
	if cfg.PushCallbackBaseURL == "" || cfg.PushCallbackToken == "" || cfg.GCPServiceAccountJSON == "" {
		logger.Info("push: scheduler not configured; reminders will not be scheduled")
		return nil
	}
	sa, err := base64.StdEncoding.DecodeString(cfg.GCPServiceAccountJSON)
	if err != nil {
		logger.Warn("push: scheduler disabled: GCP_SERVICE_ACCOUNT_JSON is not base64", slog.Any("error", err))
		return nil
	}
	client, err := cloudtasks.NewClient(cloudtasks.Config{
		ProjectID:          cfg.GCPProjectID,
		Location:           cfg.CloudTasksLocation,
		Queue:              cfg.CloudTasksQueue,
		CallbackBaseURL:    cfg.PushCallbackBaseURL,
		CallbackToken:      cfg.PushCallbackToken,
		ServiceAccountJSON: sa,
	}, &http.Client{Timeout: providerTimeout})
	if err != nil {
		logger.Warn("push: scheduler disabled: init failed", slog.Any("error", err))
		return nil
	}
	return client
}
