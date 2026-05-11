package push

import (
	"fmt"
	"log/slog"

	"github.com/gofastadev/gofasta/pkg/config"
)

// NewSender returns the Sender selected by PushConfig.Provider.
// Used by Wire DI; developers don't call this directly.
//
//   - ""    → noopSender (Send returns ErrNotConfigured; safe default)
//   - "fcm" → FCMSender (Firebase Cloud Messaging)
//
// Adding a new provider (Expo, OneSignal, APNs-direct, AWS SNS, etc.):
//
//  1. Drop a `<provider>.go` file implementing Sender into this
//     package. Mirror the structure of fcm.go.
//  2. Add a sub-config struct in pkg/config/config.go and reference it
//     from PushConfig.
//  3. Add a switch case here.
//
// No code outside pkg/push needs to change.
func NewSender(cfg *config.PushConfig, logger *slog.Logger) (Sender, error) {
	if cfg == nil {
		return &noopSender{logger: logger}, nil
	}
	switch cfg.Provider {
	case "":
		return &noopSender{logger: logger}, nil
	case "fcm":
		if cfg.FCM.CredentialsJSON == "" && cfg.FCM.CredentialsFilePath == "" {
			return nil, fmt.Errorf("push provider=fcm requires FCM.CredentialsJSON or FCM.CredentialsFilePath")
		}
		return NewFCMSender(FCMConfig{
			CredentialsJSON:     cfg.FCM.CredentialsJSON,
			CredentialsFilePath: cfg.FCM.CredentialsFilePath,
			ProjectID:           cfg.FCM.ProjectID,
		}, logger)
	default:
		return nil, fmt.Errorf("push: unknown provider %q (supported: \"\" (noop), \"fcm\")", cfg.Provider)
	}
}
