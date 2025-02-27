// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package tracing

import (
	"context"

	"github.com/mainflux/mainflux/internal/server"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/tracing"
	"go.opentelemetry.io/otel/trace"
)

// Constants to define different operations to be traced.
const (
	subscribeOP   = "receive"
	unsubscribeOp = "unsubscribe" // This is not specified in the open telemetry spec.
	processOp     = "process"
)

var _ messaging.PubSub = (*pubsubMiddleware)(nil)

type pubsubMiddleware struct {
	publisherMiddleware
	pubsub messaging.PubSub
	host   server.Config
}

// NewPubSub creates a new pubsub middleware that traces pubsub operations.
func NewPubSub(config server.Config, tracer trace.Tracer, pubsub messaging.PubSub) messaging.PubSub {
	pb := &pubsubMiddleware{
		publisherMiddleware: publisherMiddleware{
			publisher: pubsub,
			tracer:    tracer,
			host:      config,
		},
		pubsub: pubsub,
		host:   config,
	}

	return pb
}

// Subscribe creates a new subscription and traces the operation.
func (pm *pubsubMiddleware) Subscribe(ctx context.Context, id string, topic string, handler messaging.MessageHandler) error {
	ctx, span := tracing.CreateSpan(ctx, subscribeOP, id, topic, "", 0, pm.host, trace.SpanKindClient, pm.tracer)
	defer span.End()

	span.SetAttributes(defaultAttributes...)

	h := &traceHandler{
		ctx:      ctx,
		handler:  handler,
		tracer:   pm.tracer,
		host:     pm.host,
		topic:    topic,
		clientID: id,
	}

	return pm.pubsub.Subscribe(ctx, id, topic, h)
}

// Unsubscribe removes an existing subscription and traces the operation.
func (pm *pubsubMiddleware) Unsubscribe(ctx context.Context, id string, topic string) error {
	ctx, span := tracing.CreateSpan(ctx, unsubscribeOp, id, topic, "", 0, pm.host, trace.SpanKindInternal, pm.tracer)
	defer span.End()

	span.SetAttributes(defaultAttributes...)

	return pm.pubsub.Unsubscribe(ctx, id, topic)
}

// TraceHandler is used to trace the message handling operation.
type traceHandler struct {
	ctx      context.Context
	handler  messaging.MessageHandler
	tracer   trace.Tracer
	host     server.Config
	topic    string
	clientID string
}

// Handle instruments the message handling operation.
func (h *traceHandler) Handle(msg *messaging.Message) error {
	_, span := tracing.CreateSpan(h.ctx, processOp, h.clientID, h.topic, msg.Subtopic, len(msg.Payload), h.host, trace.SpanKindConsumer, h.tracer)
	defer span.End()

	span.SetAttributes(defaultAttributes...)

	return h.handler.Handle(msg)
}

// Cancel cancels the message handling operation.
func (h *traceHandler) Cancel() error {
	return h.handler.Cancel()
}
