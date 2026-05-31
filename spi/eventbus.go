package spi

import (
	"context"

	"github.com/PlugfyOS/plugfy-common/events"
)

// Handler processes a single delivered CloudEvent. Returning an error MAY
// trigger redelivery (provider-dependent: NATS JetStream NAKs the message,
// in-process bus logs and drops). Long-running handlers SHOULD respect
// ctx cancellation.
type Handler func(ctx context.Context, e events.CloudEvent) error

// Subscription represents an active consumer attached to the event bus.
// Close MUST be safe to call multiple times and SHOULD be idempotent.
type Subscription interface {
	Close() error
}

// EventBus is the SPI for the platform event plane: publish/subscribe over
// CloudEvents (the [events.CloudEvent] envelope). Implementations include the
// in-process bus (Local edition) and NATS JetStream with durable consumers
// (Enterprise edition); Kafka and Pub/Sub adapters are optional.
//
// Topic naming follows the convention `<domain>.<event>.v<major>`.
//
// Subscribe creates a durable consumer when the underlying bus supports it
// (NATS) or an in-memory subscription otherwise. The group identifier
// enables shared (load-balanced) consumption: multiple subscribers in the
// same group receive disjoint subsets of messages; different groups each
// receive a full copy.
type EventBus interface {
	Provider
	Publish(ctx context.Context, topic string, e events.CloudEvent) error
	Subscribe(ctx context.Context, topic, group string, h Handler) (Subscription, error)
}
