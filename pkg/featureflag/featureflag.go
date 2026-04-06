package featureflag

import (
	"log/slog"

	ffclient "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/ffcontext"
	"github.com/thomaspoignant/go-feature-flag/retriever/fileretriever"
)

// FeatureFlagService manages feature flags for gradual rollouts and A/B testing.
type FeatureFlagService struct {
	logger *slog.Logger
}

// NewFeatureFlagService initializes the feature flag system from a YAML config file.
func NewFeatureFlagService(configPath string, logger *slog.Logger) (*FeatureFlagService, error) {
	err := ffclient.Init(ffclient.Config{
		PollingInterval: 60,
		Retriever:       &fileretriever.Retriever{Path: configPath},
	})
	if err != nil {
		return nil, err
	}
	logger.Info("feature flags initialized", "config", configPath)
	return &FeatureFlagService{logger: logger}, nil
}

// IsEnabled checks if a feature flag is enabled for a given user context.
func (s *FeatureFlagService) IsEnabled(flagKey string, userID string, attributes map[string]interface{}) bool {
	ctx := ffcontext.NewEvaluationContextBuilder(userID).Build()
	for k, v := range attributes {
		ctx = ffcontext.NewEvaluationContextBuilder(userID).AddCustom(k, v).Build()
	}
	val, _ := ffclient.BoolVariation(flagKey, ctx, false)
	return val
}

// Close shuts down the feature flag client.
func (s *FeatureFlagService) Close() {
	ffclient.Close()
}
