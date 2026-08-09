package authn

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"

	authnauditpb "github.com/Servora-Kit/servora/api/gen/go/servora/authn/audit/v1"
	authnpb "github.com/Servora-Kit/servora/api/gen/go/servora/authn/v1"
	"github.com/Servora-Kit/servora/obs/audit"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"google.golang.org/protobuf/proto"
)

type fakeTransport struct{ operation string }

func (*fakeTransport) Kind() transport.Kind            { return transport.KindHTTP }
func (*fakeTransport) Endpoint() string                { return "" }
func (t *fakeTransport) Operation() string             { return t.operation }
func (*fakeTransport) RequestHeader() transport.Header { return fakeHeader{} }
func (*fakeTransport) ReplyHeader() transport.Header   { return fakeHeader{} }

type fakeHeader struct{}

func (fakeHeader) Get(string) string      { return "" }
func (fakeHeader) Set(string, string)     {}
func (fakeHeader) Add(string, string)     {}
func (fakeHeader) Keys() []string         { return nil }
func (fakeHeader) Values(string) []string { return nil }

type fakeAuthenticator struct {
	scheme      Scheme
	schemeCalls int
	calls       int
	loggerSets  int
	auth        Authentication
	err         error
}

func (a *fakeAuthenticator) Scheme() Scheme {
	a.schemeCalls++
	return a.scheme
}

func (a *fakeAuthenticator) Authenticate(context.Context) (Authentication, error) {
	a.calls++
	return a.auth, a.err
}

// SetLogger deliberately resembles the rejected optional-injection capability.
// Server must never discover or invoke it dynamically.
func (a *fakeAuthenticator) SetLogger(*slog.Logger) { a.loggerSets++ }

type fakeAuditor struct {
	events []cloudevents.Event
	err    error
}

func (a *fakeAuditor) Emit(_ context.Context, event cloudevents.Event) error {
	a.events = append(a.events, event)
	return a.err
}

func authnContext(operation string) context.Context {
	return transport.NewServerContext(context.Background(), &fakeTransport{operation: operation})
}

func rules(entries map[string]*authnpb.AuthnRule) func() map[string]*authnpb.AuthnRule {
	return func() map[string]*authnpb.AuthnRule { return entries }
}

func invoke(t *testing.T, mw middleware.Middleware, ctx context.Context) (context.Context, error) {
	t.Helper()
	var handlerContext context.Context
	handler := mw(func(ctx context.Context, _ any) (any, error) {
		handlerContext = ctx
		return "ok", nil
	})
	_, err := handler(ctx, nil)
	return handlerContext, err
}

func TestAuthenticatorInterfaceShape(t *testing.T) {
	typeOf := reflect.TypeOf((*Authenticator)(nil)).Elem()
	if typeOf.NumMethod() != 2 {
		t.Fatalf("Authenticator methods = %d, want 2", typeOf.NumMethod())
	}
	for _, name := range []string{"Authenticate", "Scheme"} {
		if _, ok := typeOf.MethodByName(name); !ok {
			t.Fatalf("Authenticator missing %s", name)
		}
	}
}

func TestAuthenticationContextAccessors(t *testing.T) {
	ctx := withAuthentication(context.Background(), Authentication{Subject: "user:123"})
	got, ok := AuthenticationFrom(ctx)
	if !ok || got.Subject != "user:123" {
		t.Fatalf("AuthenticationFrom = (%+v, %v), want user:123", got, ok)
	}
	subject, ok := SubjectFrom(ctx)
	if !ok || subject != "user:123" {
		t.Fatalf("SubjectFrom = (%q, %v), want user:123", subject, ok)
	}
	if _, ok := AuthenticationFrom(context.Background()); ok {
		t.Fatal("AuthenticationFrom bare context returned ok=true")
	}
}

func TestAuthenticationPublicWriterAbsent(t *testing.T) {
	body, err := os.ReadFile("context.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "func WithAuthentication(") {
		t.Fatal("public WithAuthentication must not exist")
	}
}

func TestServerConstructionValidation(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
		want string
	}{
		{name: "empty", fn: func() { Server(nil) }, want: "authenticator collection is empty"},
		{name: "nil", fn: func() { Server([]Authenticator{nil}) }, want: "authenticator[0] is nil"},
		{name: "typed nil", fn: func() { var a *fakeAuthenticator; Server([]Authenticator{a}) }, want: "authenticator[0] is nil"},
		{name: "empty scheme", fn: func() { Server([]Authenticator{&fakeAuthenticator{}}) }, want: "authenticator[0] has empty scheme"},
		{name: "duplicate scheme", fn: func() {
			Server([]Authenticator{
				&fakeAuthenticator{scheme: "jwt"},
				&fakeAuthenticator{scheme: "jwt"},
			})
		}, want: `duplicate scheme "jwt" at authenticator[1]`},
		{name: "nil option", fn: func() { Server([]Authenticator{&fakeAuthenticator{scheme: "jwt"}}, nil) }, want: "option[0] is nil"},
		{name: "unknown rule scheme", fn: func() {
			Server(
				[]Authenticator{&fakeAuthenticator{scheme: "jwt"}},
				WithRulesFuncs(rules(map[string]*authnpb.AuthnRule{
					"/svc/Op": {Mode: authnpb.AuthnRule_MODE_REQUIRED, Schemes: []string{"mtls"}},
				})),
			)
		}, want: `operation "/svc/Op" references unknown scheme "mtls"`},
		{name: "unknown public rule scheme", fn: func() {
			Server(
				[]Authenticator{&fakeAuthenticator{scheme: "jwt"}},
				WithRulesFuncs(rules(map[string]*authnpb.AuthnRule{
					"/svc/Public": {Mode: authnpb.AuthnRule_MODE_PUBLIC, Schemes: []string{"mtls"}},
				})),
			)
		}, want: `operation "/svc/Public" references unknown scheme "mtls"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := panicMessage(t, tt.fn)
			if !strings.Contains(message, tt.want) {
				t.Fatalf("panic = %q, want %q", message, tt.want)
			}
			if strings.Contains(message, "Bearer") || strings.Contains(message, "token-secret") {
				t.Fatalf("panic leaked credential: %q", message)
			}
		})
	}
}

func TestServerSnapshotsSchemesAndAuthenticatorSlice(t *testing.T) {
	first := &fakeAuthenticator{scheme: "jwt", auth: Authentication{Subject: "user:first"}}
	second := &fakeAuthenticator{scheme: "api_key", auth: Authentication{Subject: "user:second"}}
	authenticators := []Authenticator{first, second}
	mw := Server(authenticators)
	authenticators[0] = second

	for range 2 {
		ctx, err := invoke(t, mw, authnContext("/svc/Op"))
		if err != nil {
			t.Fatal(err)
		}
		subject, _ := SubjectFrom(ctx)
		if subject != "user:first" {
			t.Fatalf("subject = %q, want user:first", subject)
		}
	}
	if first.schemeCalls != 1 || second.schemeCalls != 1 {
		t.Fatalf("Scheme calls = (%d, %d), want (1, 1)", first.schemeCalls, second.schemeCalls)
	}
}

func TestServerPublicRuleSkipsAuthenticators(t *testing.T) {
	a := &fakeAuthenticator{scheme: "jwt", err: stderrors.New("must not run")}
	mw := Server(
		[]Authenticator{a},
		WithRulesFuncs(rules(map[string]*authnpb.AuthnRule{
			"/svc/Public": {Mode: authnpb.AuthnRule_MODE_PUBLIC},
		})),
	)
	if _, err := invoke(t, mw, authnContext("/svc/Public")); err != nil {
		t.Fatal(err)
	}
	if a.calls != 0 {
		t.Fatalf("calls = %d, want 0", a.calls)
	}
}

func TestServerRuleFilterPreservesInjectionOrder(t *testing.T) {
	first := &fakeAuthenticator{scheme: "jwt", auth: Authentication{Subject: "user:first"}}
	second := &fakeAuthenticator{scheme: "api_key", auth: Authentication{Subject: "user:second"}}
	mw := Server(
		[]Authenticator{first, second},
		WithRulesFuncs(rules(map[string]*authnpb.AuthnRule{
			"/svc/Op": {
				Mode:    authnpb.AuthnRule_MODE_REQUIRED,
				Schemes: []string{"api_key", "jwt"},
			},
		})),
	)
	ctx, err := invoke(t, mw, authnContext("/svc/Op"))
	if err != nil {
		t.Fatal(err)
	}
	subject, _ := SubjectFrom(ctx)
	if subject != "user:first" || second.calls != 0 {
		t.Fatalf("subject = %q, second calls = %d", subject, second.calls)
	}
}

func TestServerContinuesOnlyForNoCredentials(t *testing.T) {
	first := &fakeAuthenticator{scheme: "jwt", err: fmt.Errorf("jwt missing: %w", ErrNoCredentials)}
	second := &fakeAuthenticator{scheme: "api_key", auth: Authentication{Subject: "service:worker"}}
	mw := Server([]Authenticator{first, second})
	ctx, err := invoke(t, mw, authnContext("/svc/Op"))
	if err != nil {
		t.Fatal(err)
	}
	subject, _ := SubjectFrom(ctx)
	if subject != "service:worker" || first.calls != 1 || second.calls != 1 {
		t.Fatalf("subject=%q calls=(%d,%d)", subject, first.calls, second.calls)
	}
}

func TestServerRejectedCredentialsFailFast(t *testing.T) {
	first := &fakeAuthenticator{scheme: "jwt", err: fmt.Errorf("invalid jwt: %w", ErrCredentialsRejected)}
	second := &fakeAuthenticator{scheme: "api_key", auth: Authentication{Subject: "service:worker"}}
	_, err := invoke(t, Server([]Authenticator{first, second}), authnContext("/svc/Op"))
	if !authnpb.IsAuthnErrorReasonUnauthenticated(err) {
		t.Fatalf("error = %v, want UNAUTHENTICATED", err)
	}
	if second.calls != 0 {
		t.Fatalf("second calls = %d, want 0", second.calls)
	}
}

func TestServerErrorMappingAndCause(t *testing.T) {
	providerCause := stderrors.New("provider token-secret detail")
	tests := []struct {
		name    string
		auth    Authentication
		err     error
		matcher func(error) bool
		code    int32
		message string
	}{
		{name: "no credentials", err: fmt.Errorf("missing: %w", ErrNoCredentials), matcher: authnpb.IsAuthnErrorReasonUnauthenticated, code: 401, message: "authentication failed"},
		{name: "rejected", err: fmt.Errorf("rejected: %w: %w", ErrCredentialsRejected, providerCause), matcher: authnpb.IsAuthnErrorReasonUnauthenticated, code: 401, message: "authentication failed"},
		{name: "unavailable", err: fmt.Errorf("jwks: %w: %w", ErrUnavailable, providerCause), matcher: authnpb.IsAuthnErrorReasonUnavailable, code: 503, message: "authentication service unavailable"},
		{name: "internal", err: providerCause, matcher: authnpb.IsAuthnErrorReasonInternal, code: 500, message: "internal authentication error"},
		{name: "invalid result", auth: Authentication{}, matcher: authnpb.IsAuthnErrorReasonInternal, code: 500, message: "internal authentication error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &fakeAuthenticator{scheme: "jwt", auth: tt.auth, err: tt.err}
			_, err := invoke(t, Server([]Authenticator{a}), authnContext("/svc/Op"))
			if !tt.matcher(err) {
				t.Fatalf("error = %v, wrong generated reason", err)
			}
			ke := kerrors.FromError(err)
			if ke.Code != tt.code || ke.Message != tt.message {
				t.Fatalf("status = (%d, %q), want (%d, %q)", ke.Code, ke.Message, tt.code, tt.message)
			}
			wire, marshalErr := json.Marshal(ke.GRPCStatus().Proto())
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if bytes.Contains(wire, []byte("token-secret")) {
				t.Fatalf("wire status leaked cause: %s", wire)
			}
			httpWire, marshalErr := json.Marshal(ke)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if bytes.Contains(httpWire, []byte("token-secret")) {
				t.Fatalf("HTTP error leaked cause: %s", httpWire)
			}
			if strings.Contains(err.Error(), "token-secret") {
				t.Fatalf("error string leaked cause: %v", err)
			}
			if tt.err != nil && !stderrors.Is(err, tt.err) {
				t.Fatalf("error chain does not preserve %v", tt.err)
			}
		})
	}
}

func TestServerAuditUsesTypedSafePayloads(t *testing.T) {
	auditor := &fakeAuditor{}
	success := &fakeAuthenticator{scheme: "jwt", auth: Authentication{Subject: "user:123"}}
	ctx, err := invoke(t, Server([]Authenticator{success}, WithAuditor(auditor)), authnContext("/svc/Op"))
	if err != nil {
		t.Fatal(err)
	}
	if subject, _ := SubjectFrom(ctx); subject != "user:123" {
		t.Fatalf("subject = %q", subject)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("events = %d, want 1", len(auditor.events))
	}
	var successData authnauditpb.AuthnSuccess
	if err := proto.Unmarshal(auditor.events[0].Data(), &successData); err != nil {
		t.Fatal(err)
	}
	if successData.Scheme != "jwt" || successData.Subject != "user:123" {
		t.Fatalf("success scheme = %q, subject = %q", successData.GetScheme(), successData.GetSubject())
	}
	if auditor.events[0].Subject() != "" {
		t.Fatalf("CloudEvents subject = %q, want empty", auditor.events[0].Subject())
	}
	if _, ok := auditor.events[0].Extensions()["authid"]; ok {
		t.Fatal("authid extension must not exist")
	}

	auditor.events = nil
	failure := &fakeAuthenticator{scheme: "api_key", err: fmt.Errorf("token-secret: %w", ErrCredentialsRejected)}
	_, _ = invoke(t, Server([]Authenticator{failure}, WithAuditor(auditor)), authnContext("/svc/Op"))
	if len(auditor.events) != 1 {
		t.Fatalf("failure events = %d, want 1", len(auditor.events))
	}
	var failureData authnauditpb.AuthnFailure
	if err := proto.Unmarshal(auditor.events[0].Data(), &failureData); err != nil {
		t.Fatal(err)
	}
	if failureData.Reason != authnauditpb.AuthnFailureReason_AUTHN_FAILURE_REASON_REJECTED || failureData.Code != 401 {
		t.Fatalf("failure reason = %s, code = %d", failureData.GetReason(), failureData.GetCode())
	}
	if len(failureData.Attempts) != 1 || failureData.Attempts[0].Scheme != "api_key" {
		t.Fatalf("attempts = %+v", failureData.Attempts)
	}
	if bytes.Contains(auditor.events[0].Data(), []byte("token-secret")) {
		t.Fatal("failure audit leaked backend error")
	}
}

func TestServerAuditFailureIsBestEffort(t *testing.T) {
	auditor := &fakeAuditor{err: stderrors.New("audit unavailable")}
	a := &fakeAuthenticator{scheme: "jwt", auth: Authentication{Subject: "user:123"}}
	ctx, err := invoke(t, Server([]Authenticator{a}, WithAuditor(auditor)), authnContext("/svc/Op"))
	if err != nil {
		t.Fatal(err)
	}
	if subject, _ := SubjectFrom(ctx); subject != "user:123" {
		t.Fatalf("subject = %q", subject)
	}
}

func TestServerLoggerIsOptionalAndNotInjected(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	a := &fakeAuthenticator{scheme: "jwt", err: fmt.Errorf("token-secret: %w", ErrCredentialsRejected)}

	_, _ = invoke(t, Server([]Authenticator{a}), authnContext("/svc/Op"))
	if logs.Len() != 0 {
		t.Fatalf("logs without WithLogger = %q", logs.String())
	}

	_, _ = invoke(t, Server([]Authenticator{a}, WithLogger(logger)), authnContext("/svc/Op"))
	if !strings.Contains(logs.String(), "authentication failed") || !strings.Contains(logs.String(), "REJECTED") {
		t.Fatalf("logs = %q, want stable failure fields", logs.String())
	}
	if strings.Contains(logs.String(), "token-secret") {
		t.Fatalf("logs leaked cause: %q", logs.String())
	}

	logs.Reset()
	success := &fakeAuthenticator{scheme: "jwt", auth: Authentication{Subject: "identity-token-secret"}}
	if _, err := invoke(t, Server([]Authenticator{success}, WithLogger(logger)), authnContext("/svc/Op")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logs.String(), "identity-token-secret") {
		t.Fatalf("success log leaked subject: %q", logs.String())
	}
	if a.loggerSets != 0 {
		t.Fatalf("SetLogger calls = %d, want 0", a.loggerSets)
	}
}

func TestRemovedDispatcherFilesAreAbsent(t *testing.T) {
	for _, path := range []string{"multi.go", "subject.go", "authtype_context.go"} {
		if _, err := os.Stat(path); !stderrors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s still exists", path)
		}
	}
}

func panicMessage(t *testing.T, fn func()) (message string) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			message = fmt.Sprint(recovered)
		}
	}()
	fn()
	t.Fatal("expected panic")
	return ""
}

var _ audit.Auditor = (*fakeAuditor)(nil)
var _ Authenticator = (*fakeAuthenticator)(nil)
