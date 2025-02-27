// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"time"
)

const (
	UnpublishedEventsCheckInterval        = 1 * time.Minute
	ConnCheckInterval                     = 100 * time.Millisecond
	MaxUnpublishedEvents           uint64 = 1e6
	MaxEventStreamLen              int64  = 1e9
)

// Event represents an event.
type Event interface {
	// Encode encodes event to map.
	Encode() (map[string]interface{}, error)
}

// Publisher specifies events publishing API.
type Publisher interface {
	// Publish publishes event to stream.
	Publish(ctx context.Context, event Event) error

	// Close gracefully closes event publisher's connection.
	Close() error
}

// EventHandler represents event handler for Subscriber.
type EventHandler interface {
	// Handle handles events passed by underlying implementation.
	Handle(ctx context.Context, event Event) error
}

// Subscriber specifies event subscription API.
type Subscriber interface {
	// Subscribe subscribes to the event stream and consumes events.
	Subscribe(ctx context.Context, handler EventHandler) error

	// Close gracefully closes event subscriber's connection.
	Close() error
}
