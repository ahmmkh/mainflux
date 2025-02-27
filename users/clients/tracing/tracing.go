// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/users/clients"
	"github.com/mainflux/mainflux/users/jwt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ clients.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    clients.Service
}

// New returns a new group service with tracing capabilities.
func New(svc clients.Service, tracer trace.Tracer) clients.Service {
	return &tracingMiddleware{tracer, svc}
}

// RegisterClient traces the "RegisterClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) RegisterClient(ctx context.Context, token string, client mfclients.Client) (mfclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_register_client", trace.WithAttributes(attribute.String("identity", client.Credentials.Identity)))
	defer span.End()

	return tm.svc.RegisterClient(ctx, token, client)
}

// IssueToken traces the "IssueToken" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) IssueToken(ctx context.Context, identity, secret string) (jwt.Token, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_issue_token", trace.WithAttributes(attribute.String("identity", identity)))
	defer span.End()

	return tm.svc.IssueToken(ctx, identity, secret)
}

// RefreshToken traces the "RefreshToken" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) RefreshToken(ctx context.Context, accessToken string) (jwt.Token, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_refresh_token", trace.WithAttributes(attribute.String("access_token", accessToken)))
	defer span.End()

	return tm.svc.RefreshToken(ctx, accessToken)
}

// ViewClient traces the "ViewClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ViewClient(ctx context.Context, token string, id string) (mfclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.ViewClient(ctx, token, id)
}

// ListClients traces the "ListClients" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ListClients(ctx context.Context, token string, pm mfclients.Page) (mfclients.ClientsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_clients")
	defer span.End()

	return tm.svc.ListClients(ctx, token, pm)
}

// UpdateClient traces the "UpdateClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClient(ctx context.Context, token string, cli mfclients.Client) (mfclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_name_and_metadata", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.String("name", cli.Name),
	))
	defer span.End()

	return tm.svc.UpdateClient(ctx, token, cli)
}

// UpdateClientTags traces the "UpdateClientTags" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientTags(ctx context.Context, token string, cli mfclients.Client) (mfclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_tags", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateClientTags(ctx, token, cli)
}

// UpdateClientIdentity traces the "UpdateClientIdentity" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientIdentity(ctx context.Context, token, id, identity string) (mfclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_identity", trace.WithAttributes(
		attribute.String("id", id),
		attribute.String("identity", identity),
	))
	defer span.End()

	return tm.svc.UpdateClientIdentity(ctx, token, id, identity)
}

// UpdateClientSecret traces the "UpdateClientSecret" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientSecret(ctx context.Context, token, oldSecret, newSecret string) (mfclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_secret")
	defer span.End()

	return tm.svc.UpdateClientSecret(ctx, token, oldSecret, newSecret)
}

// GenerateResetToken traces the "GenerateResetToken" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) GenerateResetToken(ctx context.Context, email, host string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_generate_reset_token", trace.WithAttributes(
		attribute.String("email", email),
		attribute.String("host", host),
	))
	defer span.End()

	return tm.svc.GenerateResetToken(ctx, email, host)
}

// ResetSecret traces the "ResetSecret" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ResetSecret(ctx context.Context, token, secret string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_reset_secret")
	defer span.End()

	return tm.svc.ResetSecret(ctx, token, secret)
}

// SendPasswordReset traces the "SendPasswordReset" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) SendPasswordReset(ctx context.Context, host, email, user, token string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_send_password_reset", trace.WithAttributes(
		attribute.String("email", email),
		attribute.String("user", user),
	))
	defer span.End()

	return tm.svc.SendPasswordReset(ctx, host, email, user, token)
}

// ViewProfile traces the "ViewProfile" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ViewProfile(ctx context.Context, token string) (mfclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_profile")
	defer span.End()

	return tm.svc.ViewProfile(ctx, token)
}

// UpdateClientOwner traces the "UpdateClientOwner" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientOwner(ctx context.Context, token string, cli mfclients.Client) (mfclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_owner", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateClientOwner(ctx, token, cli)
}

// EnableClient traces the "EnableClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) EnableClient(ctx context.Context, token, id string) (mfclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_enable_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.EnableClient(ctx, token, id)
}

// DisableClient traces the "DisableClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) DisableClient(ctx context.Context, token, id string) (mfclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_disable_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.DisableClient(ctx, token, id)
}

// ListMembers traces the "ListMembers" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ListMembers(ctx context.Context, token, groupID string, pm mfclients.Page) (mfclients.MembersPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_members", trace.WithAttributes(attribute.String("group_id", groupID)))
	defer span.End()

	return tm.svc.ListMembers(ctx, token, groupID, pm)
}

// Identify traces the "Identify" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) Identify(ctx context.Context, token string) (string, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_identify", trace.WithAttributes(attribute.String("token", token)))
	defer span.End()

	return tm.svc.Identify(ctx, token)
}
