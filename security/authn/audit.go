package authn

import (
	"context"

	authnauditpb "github.com/Servora-Kit/servora/api/gen/go/servora/authn/audit/v1"
	"github.com/Servora-Kit/servora/obs/audit"
)

const (
	EventTypeAuthnFailure = "servora.authn.failure.v1"
	EventTypeAuthnSuccess = "servora.authn.success.v1"
)

type authenticationAttempt struct {
	scheme Scheme
	reason authnauditpb.AuthnFailureReason
}

// WithAuditor configures best-effort typed authentication decision events.
func WithAuditor(auditor audit.Auditor) Option {
	return func(cfg *serverConfig) { cfg.auditor = auditor }
}

func emitAuthnFailure(
	ctx context.Context,
	auditor audit.Auditor,
	reason authnauditpb.AuthnFailureReason,
	code int32,
	attempts []authenticationAttempt,
) {
	if auditor == nil {
		return
	}

	payloadAttempts := make([]*authnauditpb.SchemeAttempt, len(attempts))
	for index, attempt := range attempts {
		payloadAttempts[index] = &authnauditpb.SchemeAttempt{
			Scheme: string(attempt.scheme),
			Reason: attempt.reason,
		}
	}
	data := &authnauditpb.AuthnFailure{
		Reason:   reason,
		Code:     code,
		Attempts: payloadAttempts,
	}
	event := audit.NewEvent(ctx, audit.WithType(EventTypeAuthnFailure))
	if err := audit.SetProtoData(&event, data); err != nil {
		return
	}
	_ = auditor.Emit(ctx, event)
}

func emitAuthnSuccess(ctx context.Context, auditor audit.Auditor, scheme Scheme, result Authentication) {
	if auditor == nil {
		return
	}
	event := audit.NewEvent(ctx, audit.WithType(EventTypeAuthnSuccess))
	data := &authnauditpb.AuthnSuccess{Scheme: string(scheme), Subject: result.Subject}
	if err := audit.SetProtoData(&event, data); err != nil {
		return
	}
	_ = auditor.Emit(ctx, event)
}
