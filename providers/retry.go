package providers

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/pranshuj73/oni/logger"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxRetries int           // Maximum number of retry attempts (default: 3)
	BaseDelay  time.Duration // Base delay for exponential backoff (default: 1 second)
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
	}
}

// WithRetry executes a function with exponential backoff retry logic
// It returns the result of the function or the last error encountered
func WithRetry(ctx context.Context, config RetryConfig, operation string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the operation
		err := fn()
		if err == nil {
			// Success
			if attempt > 0 {
				logger.Info("Operation succeeded after retry", map[string]interface{}{
					"operation": operation,
					"attempt":   attempt + 1,
					"retries":   attempt,
				})
			}
			return nil
		}

		lastErr = err

		// Check if we should retry
		if attempt >= config.MaxRetries {
			// Max retries reached
			logger.Error("Operation failed after max retries", err, map[string]interface{}{
				"operation":  operation,
				"maxRetries": config.MaxRetries,
				"attempts":   attempt + 1,
			})
			break
		}

		// Calculate exponential backoff delay: baseDelay * 2^attempt
		backoffDelay := time.Duration(float64(config.BaseDelay) * math.Pow(2, float64(attempt)))

		logger.Warn("Operation failed, retrying with backoff", map[string]interface{}{
			"operation":    operation,
			"attempt":      attempt + 1,
			"nextAttempt":  attempt + 2,
			"backoffDelay": backoffDelay.String(),
			"error":        err.Error(),
		})

		// Wait with context cancellation support
		select {
		case <-time.After(backoffDelay):
			// Continue to next retry
		case <-ctx.Done():
			// Context cancelled
			logger.Info("Retry cancelled by context", map[string]interface{}{
				"operation": operation,
				"attempt":   attempt + 1,
			})
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// WithRetryResult is a generic version that returns a result along with error
func WithRetryResult[T any](ctx context.Context, config RetryConfig, operation string, fn func() (T, error)) (T, error) {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the operation
		res, err := fn()
		if err == nil {
			// Success
			if attempt > 0 {
				logger.Info("Operation succeeded after retry", map[string]interface{}{
					"operation": operation,
					"attempt":   attempt + 1,
					"retries":   attempt,
				})
			}
			return res, nil
		}

		lastErr = err

		// Check if we should retry
		if attempt >= config.MaxRetries {
			// Max retries reached
			logger.Error("Operation failed after max retries", err, map[string]interface{}{
				"operation":  operation,
				"maxRetries": config.MaxRetries,
				"attempts":   attempt + 1,
			})
			break
		}

		// Calculate exponential backoff delay
		backoffDelay := time.Duration(float64(config.BaseDelay) * math.Pow(2, float64(attempt)))

		logger.Warn("Operation failed, retrying with backoff", map[string]interface{}{
			"operation":    operation,
			"attempt":      attempt + 1,
			"nextAttempt":  attempt + 2,
			"backoffDelay": backoffDelay.String(),
			"error":        err.Error(),
		})

		// Wait with context cancellation support
		select {
		case <-time.After(backoffDelay):
			// Continue to next retry
		case <-ctx.Done():
			// Context cancelled
			logger.Info("Retry cancelled by context", map[string]interface{}{
				"operation": operation,
				"attempt":   attempt + 1,
			})
			var zero T
			return zero, fmt.Errorf("retry cancelled: %w", ctx.Err())
		}
	}

	var zero T
	return zero, fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// ProviderWithRetry wraps a Provider with retry logic
type ProviderWithRetry struct {
	provider Provider
	config   RetryConfig
}

// NewProviderWithRetry creates a new provider with retry logic
func NewProviderWithRetry(provider Provider, config RetryConfig) *ProviderWithRetry {
	return &ProviderWithRetry{
		provider: provider,
		config:   config,
	}
}

// Name returns the provider name
func (p *ProviderWithRetry) Name() string {
	return p.provider.Name()
}

// GetEpisodeInfo wraps the provider's GetEpisodeInfo with retry logic
func (p *ProviderWithRetry) GetEpisodeInfo(ctx context.Context, mediaID int, episodeNum int, title string) (*EpisodeInfo, error) {
	operation := fmt.Sprintf("%s.GetEpisodeInfo(mediaID=%d, episode=%d)", p.provider.Name(), mediaID, episodeNum)

	return WithRetryResult(ctx, p.config, operation, func() (*EpisodeInfo, error) {
		return p.provider.GetEpisodeInfo(ctx, mediaID, episodeNum, title)
	})
}

// GetVideoLink wraps the provider's GetVideoLink with retry logic
func (p *ProviderWithRetry) GetVideoLink(ctx context.Context, episodeInfo *EpisodeInfo, quality string, subOrDub string) (*VideoData, error) {
	operation := fmt.Sprintf("%s.GetVideoLink(quality=%s, subOrDub=%s)", p.provider.Name(), quality, subOrDub)

	return WithRetryResult(ctx, p.config, operation, func() (*VideoData, error) {
		return p.provider.GetVideoLink(ctx, episodeInfo, quality, subOrDub)
	})
}
