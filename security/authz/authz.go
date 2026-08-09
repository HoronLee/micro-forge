// Package authz provides engine-neutral authorization contracts and Kratos
// server middleware.
package authz

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"reflect"

	authnauditpb "github.com/Servora-Kit/servora/api/gen/go/servora/authn/audit/v1"
	authnpb "github.com/Servora-Kit/servora/api/gen/go/servora/authn/v1"
	authzauditpb "github.com/Servora-Kit/servora/api/gen/go/servora/authz/audit/v1"
	authzpb "github.com/Servora-Kit/servora/api/gen/go/servora/authz/v1"
	"github.com/Servora-Kit/servora/obs/audit"
	"github.com/Servora-Kit/servora/security/authn"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
)

// Resource identifies one authorization target.
type Resource struct {
	Type string
	ID   string
}

// CheckRequest describes one engine-neutral authorization decision.
// Subject must be a stable, non-replayable identity identifier, never a credential.
type CheckRequest struct {
	Subject    string
	Action     string
	Resource   Resource
	Attributes map[string]any
}

// Authorizer is the single-method authorization contract.
type Authorizer interface {
	Check(context.Context, CheckRequest) (bool, error)
}

// BatchAuthorizer is the optional ordered batch-check capability.
type BatchAuthorizer interface {
	Authorizer
	BatchCheck(context.Context, []CheckRequest) ([]bool, error)
}

// Lister is the optional capability for listing allowed resource IDs.
type Lister interface {
	Authorizer
	ListAllowed(ctx context.Context, subject, action, resourceType string) ([]string, error)
}

// ErrUnavailable identifies a temporarily unavailable authorization dependency.
var ErrUnavailable = stderrors.New("authz: unavailable")

var (
	errMissingSubject   = stderrors.New("authz: trusted subject is missing")
	errMissingTransport = stderrors.New("authz: server transport is missing")
)

// Option configures Server.
type Option func(*serverConfig)

type serverConfig struct {
	rules       map[string]*authzpb.AuthzRule
	subjectFunc func(context.Context) (string, bool)
	auditor     audit.Auditor
	logger      *slog.Logger
}

type compiledRule struct {
	mode            authzpb.AuthzMode
	action          string
	resourceType    string
	resourceIDField string
}

// WithRulesFuncs merges generated rule tables. Later entries win.
func WithRulesFuncs(fns ...func() map[string]*authzpb.AuthzRule) Option {
	return func(cfg *serverConfig) {
		for _, fn := range fns {
			if fn == nil {
				continue
			}
			for operation, rule := range fn() {
				if rule == nil {
					continue
				}
				if cfg.rules == nil {
					cfg.rules = make(map[string]*authzpb.AuthzRule)
				}
				cfg.rules[operation] = rule
			}
		}
	}
}

// WithSubjectFunc overrides the standard authn.SubjectFrom reader.
func WithSubjectFunc(fn func(context.Context) (string, bool)) Option {
	return func(cfg *serverConfig) { cfg.subjectFunc = fn }
}

// WithLogger configures root middleware diagnostics. It does not inject the
// logger into Authorizer implementations.
func WithLogger(logger *slog.Logger) Option {
	return func(cfg *serverConfig) { cfg.logger = logger }
}

// Server constructs authorization middleware and validates static wiring
// before returning it.
func Server(authorizer Authorizer, opts ...Option) middleware.Middleware {
	if isNilAuthorizer(authorizer) {
		panic("authz: authorizer is nil")
	}

	cfg := &serverConfig{subjectFunc: authn.SubjectFrom}
	for index, opt := range opts {
		if opt == nil {
			panic(fmt.Sprintf("authz: option[%d] is nil", index))
		}
		opt(cfg)
		if cfg.subjectFunc == nil {
			panic(fmt.Sprintf("authz: option[%d] set subject function to nil", index))
		}
	}
	compiledRules := compileRules(cfg.rules)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok || tr == nil {
				logAuthorization(ctx, cfg.logger, slog.LevelError, "", CheckRequest{}, authzauditpb.AuthzDecision_REASON_INTERNAL)
				return nil, authzpb.ErrorAuthzErrorReasonInternal("internal authorization error").WithCause(concealAuthorizationCause(errMissingTransport))
			}

			operation := tr.Operation()
			rule, found := compiledRules[operation]
			if !found {
				cause := fmt.Errorf("authz: no authorization rule for operation %q", operation)
				logAuthorization(ctx, cfg.logger, slog.LevelError, operation, CheckRequest{}, authzauditpb.AuthzDecision_REASON_INTERNAL)
				return nil, authzpb.ErrorAuthzErrorReasonInternal("internal authorization error").WithCause(concealAuthorizationCause(cause))
			}
			if rule.mode == authzpb.AuthzMode_AUTHZ_MODE_NONE {
				return handler(ctx, req)
			}

			subject, ok := cfg.subjectFunc(ctx)
			if !ok || subject == "" {
				logAuthorization(ctx, cfg.logger, slog.LevelWarn, operation, CheckRequest{}, authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_NO_CREDENTIALS)
				return nil, authnpb.ErrorAuthnErrorReasonUnauthenticated("authentication failed").WithCause(concealAuthorizationCause(errMissingSubject))
			}

			checkRequest := CheckRequest{
				Subject: subject,
				Action:  rule.action,
				Resource: Resource{
					Type: rule.resourceType,
				},
			}
			resource, err := resolveResource(rule, req)
			if err != nil {
				checkRequest.Resource = resource
				emitAuthzDecision(ctx, cfg.auditor, checkRequest, authzauditpb.AuthzDecision_DECISION_ERROR, authzauditpb.AuthzDecision_REASON_INVALID_REQUEST, 400)
				logAuthorization(ctx, cfg.logger, slog.LevelWarn, operation, checkRequest, authzauditpb.AuthzDecision_REASON_INVALID_REQUEST)
				return nil, authzpb.ErrorAuthzErrorReasonInvalidRequest("invalid authorization request").WithCause(concealAuthorizationCause(err))
			}
			checkRequest.Resource = resource

			allowed, checkErr := authorizer.Check(ctx, checkRequest)
			if checkErr != nil {
				reason := authzauditpb.AuthzDecision_REASON_INTERNAL
				code := int32(500)
				if stderrors.Is(checkErr, ErrUnavailable) {
					reason = authzauditpb.AuthzDecision_REASON_UNAVAILABLE
					code = 503
				}
				emitAuthzDecision(ctx, cfg.auditor, checkRequest, authzauditpb.AuthzDecision_DECISION_ERROR, reason, code)
				logAuthorization(ctx, cfg.logger, slog.LevelError, operation, checkRequest, reason)
				return nil, authorizationError(reason, checkErr)
			}
			if !allowed {
				emitAuthzDecision(ctx, cfg.auditor, checkRequest, authzauditpb.AuthzDecision_DECISION_DENIED, authzauditpb.AuthzDecision_REASON_DENIED, 403)
				logAuthorization(ctx, cfg.logger, slog.LevelWarn, operation, checkRequest, authzauditpb.AuthzDecision_REASON_DENIED)
				return nil, authzpb.ErrorAuthzErrorReasonDenied("permission denied")
			}

			emitAuthzDecision(ctx, cfg.auditor, checkRequest, authzauditpb.AuthzDecision_DECISION_ALLOWED, authzauditpb.AuthzDecision_REASON_ALLOWED, 200)
			logAuthorization(ctx, cfg.logger, slog.LevelInfo, operation, checkRequest, authzauditpb.AuthzDecision_REASON_ALLOWED)
			return handler(ctx, req)
		}
	}
}

func compileRules(rules map[string]*authzpb.AuthzRule) map[string]compiledRule {
	compiled := make(map[string]compiledRule, len(rules))
	for operation, rule := range rules {
		compiled[operation] = compiledRule{
			mode:            rule.GetMode(),
			action:          rule.GetAction(),
			resourceType:    rule.GetResourceType(),
			resourceIDField: rule.GetResourceIdField(),
		}
	}
	return compiled
}

func isNilAuthorizer(authorizer Authorizer) bool {
	if authorizer == nil {
		return true
	}
	value := reflect.ValueOf(authorizer)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func authorizationError(reason authzauditpb.AuthzDecision_Reason, cause error) error {
	concealed := concealAuthorizationCause(cause)
	switch reason {
	case authzauditpb.AuthzDecision_REASON_UNAVAILABLE:
		return authzpb.ErrorAuthzErrorReasonUnavailable("authorization service unavailable").WithCause(concealed)
	default:
		return authzpb.ErrorAuthzErrorReasonInternal("internal authorization error").WithCause(concealed)
	}
}

type concealedAuthorizationCause struct {
	cause error
}

func (e concealedAuthorizationCause) Error() string { return "authorization cause withheld" }
func (e concealedAuthorizationCause) Unwrap() error { return e.cause }

func concealAuthorizationCause(cause error) error {
	if cause == nil {
		return nil
	}
	return concealedAuthorizationCause{cause: cause}
}

func logAuthorization(
	ctx context.Context,
	logger *slog.Logger,
	level slog.Level,
	operation string,
	req CheckRequest,
	reason fmt.Stringer,
) {
	if logger == nil {
		return
	}
	logger.LogAttrs(ctx, level, "authorization decision",
		slog.String("operation", operation),
		slog.String("action", req.Action),
		slog.String("resource_type", req.Resource.Type),
		slog.String("reason", reason.String()),
	)
}
