package ui

import "time"

// ToastMsg requests a transient message in the app footer.
type ToastMsg struct {
	Text     string
	Duration time.Duration
}

// ClearToastMsg clears a toast by id.
type ClearToastMsg struct {
	ID int
}
