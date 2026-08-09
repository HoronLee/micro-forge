package authz

import (
	"context"

	authzauditpb "github.com/Servora-Kit/servora/api/gen/go/servora/authz/audit/v1"
	"github.com/Servora-Kit/servora/obs/audit"
)

const (
	EventTypeAuthzAllowed = "servora.authz.allowed.v1"
	EventTypeAuthzDenied  = "servora.authz.denied.v1"
	EventTypeAuthzError   = "servora.authz.error.v1"
)

// WithAuditor configures best-effort typed authorization decision events.
func WithAuditor(auditor audit.Auditor) Option {
	return func(cfg *serverConfig) { cfg.auditor = auditor }
}

func emitAuthzDecision(
	ctx context.Context,
	auditor audit.Auditor,
	req CheckRequest,
	decision authzauditpb.AuthzDecision_Decision,
	reason authzauditpb.AuthzDecision_Reason,
	code int32,
) {
	if auditor == nil {
		return
	}

	eventType := EventTypeAuthzError
	switch decision {
	case authzauditpb.AuthzDecision_DECISION_ALLOWED:
		eventType = EventTypeAuthzAllowed
	case authzauditpb.AuthzDecision_DECISION_DENIED:
		eventType = EventTypeAuthzDenied
	}
	event := audit.NewEvent(ctx,
		audit.WithType(eventType),
		audit.WithSubject(req.Resource.Type+":"+req.Resource.ID),
	)
	data := &authzauditpb.AuthzDecision{
		Decision:     decision,
		Subject:      req.Subject,
		Action:       req.Action,
		ResourceType: req.Resource.Type,
		ResourceId:   req.Resource.ID,
		Reason:       reason,
		Code:         code,
	}
	if err := audit.SetProtoData(&event, data); err != nil {
		return
	}
	_ = auditor.Emit(ctx, event)
}
