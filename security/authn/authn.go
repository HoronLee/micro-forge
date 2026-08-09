// Package authn provides engine-neutral authentication contracts and Kratos
// server middleware. Credential extraction and provider behavior belong to
// application or contrib Authenticator implementations.
package authn

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"

	authnauditpb "github.com/Servora-Kit/servora/api/gen/go/servora/authn/audit/v1"
	authnpb "github.com/Servora-Kit/servora/api/gen/go/servora/authn/v1"
	"github.com/Servora-Kit/servora/obs/audit"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
)

// Scheme is an open authentication mechanism name owned by an Authenticator.
type Scheme string

// Authentication is the validated result exposed to downstream middleware.
// Subject must be a stable, non-replayable identity identifier, never a credential.
type Authentication struct {
	Subject string
}

// Authenticator validates one credential mechanism.
type Authenticator interface {
	Scheme() Scheme
	Authenticate(context.Context) (Authentication, error)
}

var (
	// ErrNoCredentials means the Authenticator's credential carrier is absent.
	ErrNoCredentials = errors.New("authn: no credentials")
	// ErrCredentialsRejected means credentials were present but invalid.
	ErrCredentialsRejected = errors.New("authn: credentials rejected")
	// ErrUnavailable means an authentication dependency is temporarily unavailable.
	ErrUnavailable = errors.New("authn: unavailable")

	errInvalidAuthenticationResult = errors.New("authn: successful result has empty subject")
)

// Option configures Server.
type Option func(*serverConfig)

type serverConfig struct {
	rules   map[string]*authnpb.AuthnRule
	auditor audit.Auditor
	logger  *slog.Logger
}

type compiledAuthenticator struct {
	scheme        Scheme
	authenticator Authenticator
}

type compiledRule struct {
	public     bool
	candidates []int
}

// WithRulesFuncs merges generated AuthN rule tables. Later entries win.
func WithRulesFuncs(fns ...func() map[string]*authnpb.AuthnRule) Option {
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
					cfg.rules = make(map[string]*authnpb.AuthnRule)
				}
				cfg.rules[operation] = rule
			}
		}
	}
}

// WithLogger configures root middleware diagnostics. It does not inject the
// logger into Authenticator implementations.
func WithLogger(logger *slog.Logger) Option {
	return func(cfg *serverConfig) { cfg.logger = logger }
}

// Server constructs authentication middleware and validates all static wiring
// before returning it.
func Server(authenticators []Authenticator, opts ...Option) middleware.Middleware {
	cfg := &serverConfig{}
	for index, opt := range opts {
		if opt == nil {
			panic(fmt.Sprintf("authn: option[%d] is nil", index))
		}
		opt(cfg)
	}

	entries := compileAuthenticators(authenticators)
	compiledRules := compileAuthnRules(cfg.rules, entries)
	allCandidates := make([]int, len(entries))
	for index := range entries {
		allCandidates[index] = index
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			operation := ""
			if tr, ok := transport.FromServerContext(ctx); ok && tr != nil {
				operation = tr.Operation()
			}

			candidates := allCandidates
			if rule, ok := compiledRules[operation]; ok {
				if rule.public {
					return handler(ctx, req)
				}
				candidates = rule.candidates
			}

			attempts := make([]authenticationAttempt, 0, len(candidates))
			var lastNoCredentials error
			for _, candidate := range candidates {
				entry := entries[candidate]
				result, err := entry.authenticator.Authenticate(ctx)
				if err == nil {
					if result.Subject == "" {
						attempts = append(attempts, authenticationAttempt{scheme: entry.scheme, reason: authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_INVALID_RESULT})
						return nil, finishAuthenticationFailure(ctx, cfg, entry.scheme, authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_INVALID_RESULT, 500, attempts, errInvalidAuthenticationResult)
					}

					nextCtx := withAuthentication(ctx, result)
					emitAuthnSuccess(nextCtx, cfg.auditor, entry.scheme, result)
					logAuthenticationSuccess(nextCtx, cfg.logger, entry.scheme)
					return handler(nextCtx, req)
				}

				switch {
				case errors.Is(err, ErrNoCredentials):
					lastNoCredentials = err
					attempts = append(attempts, authenticationAttempt{scheme: entry.scheme, reason: authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_NO_CREDENTIALS})
					continue
				case errors.Is(err, ErrCredentialsRejected):
					attempts = append(attempts, authenticationAttempt{scheme: entry.scheme, reason: authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_REJECTED})
					return nil, finishAuthenticationFailure(ctx, cfg, entry.scheme, authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_REJECTED, 401, attempts, err)
				case errors.Is(err, ErrUnavailable):
					attempts = append(attempts, authenticationAttempt{scheme: entry.scheme, reason: authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_UNAVAILABLE})
					return nil, finishAuthenticationFailure(ctx, cfg, entry.scheme, authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_UNAVAILABLE, 503, attempts, err)
				default:
					attempts = append(attempts, authenticationAttempt{scheme: entry.scheme, reason: authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_INTERNAL})
					return nil, finishAuthenticationFailure(ctx, cfg, entry.scheme, authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_INTERNAL, 500, attempts, err)
				}
			}

			if lastNoCredentials == nil {
				lastNoCredentials = ErrNoCredentials
			}
			return nil, finishAuthenticationFailure(ctx, cfg, "", authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_NO_CREDENTIALS, 401, attempts, lastNoCredentials)
		}
	}
}

func compileAuthenticators(authenticators []Authenticator) []compiledAuthenticator {
	if len(authenticators) == 0 {
		panic("authn: authenticator collection is empty")
	}

	copied := append([]Authenticator(nil), authenticators...)
	entries := make([]compiledAuthenticator, len(copied))
	seen := make(map[Scheme]int, len(copied))
	for index, authenticator := range copied {
		if isNilAuthenticator(authenticator) {
			panic(fmt.Sprintf("authn: authenticator[%d] is nil", index))
		}
		scheme := authenticator.Scheme()
		if scheme == "" {
			panic(fmt.Sprintf("authn: authenticator[%d] has empty scheme", index))
		}
		if _, exists := seen[scheme]; exists {
			panic(fmt.Sprintf("authn: duplicate scheme %q at authenticator[%d]", scheme, index))
		}
		seen[scheme] = index
		entries[index] = compiledAuthenticator{scheme: scheme, authenticator: authenticator}
	}
	return entries
}

func compileAuthnRules(rules map[string]*authnpb.AuthnRule, entries []compiledAuthenticator) map[string]compiledRule {
	if len(rules) == 0 {
		return nil
	}

	indexes := make(map[Scheme]int, len(entries))
	for index, entry := range entries {
		indexes[entry.scheme] = index
	}

	compiled := make(map[string]compiledRule, len(rules))
	for operation, rule := range rules {
		schemes := rule.GetSchemes()
		for _, rawScheme := range schemes {
			scheme := Scheme(rawScheme)
			if _, ok := indexes[scheme]; !ok {
				panic(fmt.Sprintf("authn: operation %q references unknown scheme %q", operation, scheme))
			}
		}
		if rule.GetMode() == authnpb.AuthnRule_MODE_PUBLIC {
			compiled[operation] = compiledRule{public: true}
			continue
		}

		// An empty scheme list allows all configured authenticators.
		if len(schemes) == 0 {
			candidates := make([]int, len(entries))
			for index := range entries {
				candidates[index] = index
			}
			compiled[operation] = compiledRule{candidates: candidates}
			continue
		}

		allowed := make(map[Scheme]struct{}, len(schemes))
		for _, rawScheme := range schemes {
			allowed[Scheme(rawScheme)] = struct{}{}
		}
		candidates := make([]int, 0, len(allowed))
		for index, entry := range entries {
			if _, ok := allowed[entry.scheme]; ok {
				candidates = append(candidates, index)
			}
		}
		compiled[operation] = compiledRule{candidates: candidates}
	}
	return compiled
}

func isNilAuthenticator(authenticator Authenticator) bool {
	if authenticator == nil {
		return true
	}
	value := reflect.ValueOf(authenticator)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func finishAuthenticationFailure(
	ctx context.Context,
	cfg *serverConfig,
	scheme Scheme,
	reason authnauditpb.AuthnFailureReason,
	code int32,
	attempts []authenticationAttempt,
	cause error,
) error {
	emitAuthnFailure(ctx, cfg.auditor, reason, code, attempts)
	logAuthenticationFailure(ctx, cfg.logger, scheme, reason)
	return authenticationError(reason, cause)
}

func authenticationError(reason authnauditpb.AuthnFailureReason, cause error) error {
	concealed := concealAuthenticationCause(cause)
	switch reason {
	case authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_NO_CREDENTIALS,
		authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_REJECTED:
		return authnpb.ErrorAuthnErrorReasonUnauthenticated("authentication failed").WithCause(concealed)
	case authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_UNAVAILABLE:
		return authnpb.ErrorAuthnErrorReasonUnavailable("authentication service unavailable").WithCause(concealed)
	default:
		return authnpb.ErrorAuthnErrorReasonInternal("internal authentication error").WithCause(concealed)
	}
}

type concealedAuthenticationCause struct {
	cause error
}

func (e concealedAuthenticationCause) Error() string { return "authentication cause withheld" }
func (e concealedAuthenticationCause) Unwrap() error { return e.cause }

func concealAuthenticationCause(cause error) error {
	if cause == nil {
		return nil
	}
	return concealedAuthenticationCause{cause: cause}
}

func logAuthenticationSuccess(ctx context.Context, logger *slog.Logger, scheme Scheme) {
	if logger == nil {
		return
	}
	logger.InfoContext(ctx, "authentication succeeded", "scheme", scheme)
}

func logAuthenticationFailure(ctx context.Context, logger *slog.Logger, scheme Scheme, reason authnauditpb.AuthnFailureReason) {
	if logger == nil {
		return
	}
	logger.ErrorContext(ctx, "authentication failed", "scheme", scheme, "reason", reason.String())
}
