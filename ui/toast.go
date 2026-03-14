package ui

import "time"

const DefaultToastDuration = 1 * time.Second

// ToastKind defines the style/duration behavior for a toast.
type ToastKind int

const (
	ToastInfo ToastKind = iota
	ToastSuccess
	ToastError
)

// ToastMsg requests a transient message in the app footer.
type ToastMsg struct {
	Text     string
	Duration time.Duration
	Kind     ToastKind
}

// ClearToastMsg clears a toast by id.
type ClearToastMsg struct {
	ID int
}
