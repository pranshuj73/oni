package discord

import (
	"fmt"
	"time"

	"github.com/hugolgst/rich-go/client"
)

// PresenceManager manages Discord Rich Presence
type PresenceManager struct {
	enabled    bool
	connected  bool
}

// NewPresenceManager creates a new presence manager
func NewPresenceManager(enabled bool) *PresenceManager {
	return &PresenceManager{
		enabled:   enabled,
		connected: false,
	}
}

// Connect connects to Discord
func (pm *PresenceManager) Connect() error {
	if !pm.enabled {
		return nil
	}

	if pm.connected {
		return nil
	}

	err := client.Login("1436820992306450532") // You should use your own Discord application ID
	if err != nil {
		// Don't error if Discord is not running
		return nil
	}

	pm.connected = true
	return nil
}

// SetPresence sets the Discord Rich Presence
func (pm *PresenceManager) SetPresence(title string, episode int, year int, coverURL string) error {
	if !pm.enabled {
		return nil
	}

	// Ensure we're connected
	if !pm.connected {
		if err := pm.Connect(); err != nil {
			return nil // Silently fail if Discord is not running
		}
	}

	now := time.Now()
	activity := client.Activity{
		Details:    fmt.Sprintf("Watching %s", title),
		State:      fmt.Sprintf("Episode %d", episode),
		LargeImage: coverURL,
		LargeText:  title,
		Timestamps: &client.Timestamps{
			Start: &now,
		},
	}

	err := client.SetActivity(activity)
	if err != nil {
		// Silently fail if Discord connection is lost
		pm.connected = false
		return nil
	}

	return nil
}

// Clear clears the Discord Rich Presence
func (pm *PresenceManager) Clear() error {
	if !pm.enabled || !pm.connected {
		return nil
	}

	// Ignore errors from Logout (e.g., broken pipe if Discord closed)
	client.Logout()
	pm.connected = false
	return nil
}

// IsEnabled returns whether Discord presence is enabled
func (pm *PresenceManager) IsEnabled() bool {
	return pm.enabled
}

// IsConnected returns whether Discord is connected
func (pm *PresenceManager) IsConnected() bool {
	return pm.connected
}

