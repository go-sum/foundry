package notification

import "time"

// Severity classifies the urgency of a notification.
type Severity int

const (
	SeverityInfo     Severity = iota
	SeverityWarning
	SeverityCritical
)

// Channel identifies a delivery mechanism.
type Channel string

const (
	ChannelEmail   Channel = "email"
	ChannelWebhook Channel = "webhook"
	ChannelLog     Channel = "log"
)

// Notification is the unit of work sent through the system.
type Notification struct {
	ID          string
	Severity    Severity
	Channels    []Channel         // empty = all configured channels
	Subject     string
	Body        string
	Metadata    map[string]string // channel-specific: "to", "from", "url", "html"
	Correlation Correlation
	Timestamp   time.Time         // zero = set by Dispatcher.Send
}

// Correlation carries request-scoped fields from the error handling guide (section 6.2).
type Correlation struct {
	RequestID string
	TraceID   string
	SpanID    string
	Op        string
	Subsystem string
	DedupeKey string
	Env       string
}
