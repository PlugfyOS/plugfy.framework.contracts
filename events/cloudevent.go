// Package events defines the CloudEvents 1.0 envelope used across the Plugfy
// event bus (NATS JetStream in production, in-process locally) and the canonical
// event type constants. The bus SPI that carries these is [spi.EventBus]; the
// platform layer model lives in PlugfyOS/plugfy-platform.
package events

import (
	"encoding/json"
	"time"
)

// SpecVersion is the CloudEvents spec version emitted by Plugfy.
const SpecVersion = "1.0"

// CloudEvent is a CloudEvents 1.0 (JSON mode) envelope.
type CloudEvent struct {
	SpecVersion     string          `json:"specversion"`
	ID              string          `json:"id"`
	Source          string          `json:"source"`
	Type            string          `json:"type"`
	Subject         string          `json:"subject,omitempty"`
	Time            time.Time       `json:"time"`
	DataContentType string          `json:"datacontenttype,omitempty"`
	Data            json.RawMessage `json:"data,omitempty"`
}

// New builds a CloudEvent with the standard envelope fields populated.
// data is marshaled to JSON; a nil data is allowed.
func New(id, source, eventType, subject string, data any) (CloudEvent, error) {
	var raw json.RawMessage
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return CloudEvent{}, err
		}
		raw = b
	}
	return CloudEvent{
		SpecVersion:     SpecVersion,
		ID:              id,
		Source:          source,
		Type:            eventType,
		Subject:         subject,
		Time:            time.Now().UTC(),
		DataContentType: "application/json",
		Data:            raw,
	}, nil
}

// Canonical event types emitted across the platform event bus.
const (
	TypeIAMOrgCreated      = "com.plugfy.iam.org.created"
	TypeIAMGrantChanged    = "com.plugfy.iam.grant.changed"
	TypeIAMUserInvited     = "com.plugfy.iam.user.invited"
	TypeRuntimeScheduled   = "com.plugfy.runtime.instance.scheduled"
	TypeRuntimeReady       = "com.plugfy.runtime.instance.ready"
	TypeRuntimeIdle        = "com.plugfy.runtime.instance.idle"
	TypeRuntimeStopped     = "com.plugfy.runtime.instance.stopped"
	TypeAgentRunStarted    = "com.plugfy.agent.run.started"
	TypeAgentRunStep       = "com.plugfy.agent.run.step"
	TypeAgentRunCompleted  = "com.plugfy.agent.run.completed"
	TypeGuardrailTriggered = "com.plugfy.agent.guardrail.triggered"
	TypeMarketplaceInstall = "com.plugfy.marketplace.app.installed"
	TypeJobEnqueued        = "com.plugfy.jobs.job.enqueued"
	TypeJobStarted         = "com.plugfy.jobs.job.started"
	TypeJobSucceeded       = "com.plugfy.jobs.job.succeeded"
	TypeJobFailed          = "com.plugfy.jobs.job.failed"
	TypeNotificationSent   = "com.plugfy.notifications.sent"
	TypePlatformAudit      = "com.plugfy.platform.audit"
)
