package authz

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	authnpb "github.com/Servora-Kit/servora/api/gen/go/servora/authn/v1"
	authzauditpb "github.com/Servora-Kit/servora/api/gen/go/servora/authz/audit/v1"
	authzpb "github.com/Servora-Kit/servora/api/gen/go/servora/authz/v1"
	"github.com/Servora-Kit/servora/security/authn"
)

const testOperation = "/test.v1.ResourceService/Get"

type fakeTransport struct{ operation string }

func (f *fakeTransport) Kind() transport.Kind            { return transport.KindHTTP }
func (f *fakeTransport) Endpoint() string                { return "" }
func (f *fakeTransport) Operation() string               { return f.operation }
func (f *fakeTransport) RequestHeader() transport.Header { return fakeHeader{} }
func (f *fakeTransport) ReplyHeader() transport.Header   { return fakeHeader{} }

type fakeHeader struct{}

func (fakeHeader) Get(string) string      { return "" }
func (fakeHeader) Set(string, string)     {}
func (fakeHeader) Add(string, string)     {}
func (fakeHeader) Keys() []string         { return nil }
func (fakeHeader) Values(string) []string { return nil }

func serverContext(operation string) context.Context {
	return transport.NewServerContext(context.Background(), &fakeTransport{operation: operation})
}

type fakeAuthorizer struct {
	mu         sync.Mutex
	allowed    bool
	err        error
	check      func(context.Context, CheckRequest) (bool, error)
	requests   []CheckRequest
	setLoggers int
}

func (f *fakeAuthorizer) Check(ctx context.Context, req CheckRequest) (bool, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()
	if f.check != nil {
		return f.check(ctx, req)
	}
	return f.allowed, f.err
}

func (f *fakeAuthorizer) SetLogger(*slog.Logger) {
	f.mu.Lock()
	f.setLoggers++
	f.mu.Unlock()
}

func (f *fakeAuthorizer) requestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fakeAuthorizer) lastRequest() CheckRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[len(f.requests)-1]
}

type capabilityAuthorizer struct{ fakeAuthorizer }

func (*capabilityAuthorizer) BatchCheck(context.Context, []CheckRequest) ([]bool, error) {
	return []bool{}, nil
}

func (*capabilityAuthorizer) ListAllowed(context.Context, string, string, string) ([]string, error) {
	return []string{}, nil
}

var (
	_ Authorizer      = (*fakeAuthorizer)(nil)
	_ BatchAuthorizer = (*capabilityAuthorizer)(nil)
	_ Lister          = (*capabilityAuthorizer)(nil)
)

type staticAuthenticator struct{ subject string }

func (staticAuthenticator) Scheme() authn.Scheme { return "test" }
func (a staticAuthenticator) Authenticate(context.Context) (authn.Authentication, error) {
	return authn.Authentication{Subject: a.subject}, nil
}

type captureAuditor struct {
	mu     sync.Mutex
	events []cloudevents.Event
	err    error
}

func (a *captureAuditor) Emit(_ context.Context, event cloudevents.Event) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, event)
	return a.err
}

func (a *captureAuditor) all() []cloudevents.Event {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]cloudevents.Event(nil), a.events...)
}

func checkRule() *authzpb.AuthzRule {
	return &authzpb.AuthzRule{
		Mode:            authzpb.AuthzMode_AUTHZ_MODE_CHECK,
		Action:          "read",
		ResourceType:    "document",
		ResourceIdField: "value",
	}
}

func rules(rule *authzpb.AuthzRule) Option {
	return WithRulesFuncs(func() map[string]*authzpb.AuthzRule {
		return map[string]*authzpb.AuthzRule{testOperation: rule}
	})
}

func subject(subject string) Option {
	return WithSubjectFunc(func(context.Context) (string, bool) {
		return subject, subject != ""
	})
}

func invoke(t *testing.T, mw middleware.Middleware, ctx context.Context, req any, handler middleware.Handler) (any, error) {
	t.Helper()
	if handler == nil {
		handler = func(context.Context, any) (any, error) { return "ok", nil }
	}
	return mw(handler)(ctx, req)
}

func assertPanicContains(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil || !strings.Contains(recovered.(string), want) {
			t.Fatalf("panic = %v, want substring %q", recovered, want)
		}
	}()
	fn()
}

func TestServerConstructionValidation(t *testing.T) {
	assertPanicContains(t, "authorizer is nil", func() { Server(nil) })
	var typedNil *fakeAuthorizer
	assertPanicContains(t, "authorizer is nil", func() { Server(typedNil) })
	assertPanicContains(t, "option[0] is nil", func() { Server(&fakeAuthorizer{}, nil) })
	assertPanicContains(t, "option[0]", func() {
		Server(&fakeAuthorizer{}, WithSubjectFunc(nil))
	})
}

func TestServerNoTransportFailsClosed(t *testing.T) {
	called := false
	_, err := invoke(t, Server(&fakeAuthorizer{}), context.Background(), nil, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})
	if !authzpb.IsAuthzErrorReasonInternal(err) || called {
		t.Fatalf("called = %v, error = %v", called, err)
	}
}

func TestServerModeNonePassesThrough(t *testing.T) {
	called := false
	_, err := invoke(t, Server(&fakeAuthorizer{}, rules(&authzpb.AuthzRule{Mode: authzpb.AuthzMode_AUTHZ_MODE_NONE})), serverContext(testOperation), nil, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})
	if err != nil || !called {
		t.Fatalf("called = %v, error = %v", called, err)
	}
}

func TestServerMissingRuleFailsClosed(t *testing.T) {
	_, err := invoke(t, Server(&fakeAuthorizer{}), serverContext(testOperation), nil, nil)
	if !authzpb.IsAuthzErrorReasonInternal(err) {
		t.Fatalf("error = %v, want generated INTERNAL", err)
	}
}

func TestServerUsesStandardAuthnSubjectByDefault(t *testing.T) {
	authorizer := &fakeAuthorizer{allowed: true}
	authzMiddleware := Server(authorizer, rules(checkRule()))
	authnMiddleware := authn.Server([]authn.Authenticator{staticAuthenticator{subject: "user:alice"}})
	called := false
	chain := authnMiddleware(authzMiddleware(func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	}))
	_, err := chain(serverContext(testOperation), &wrapperspb.StringValue{Value: "doc-1"})
	if err != nil || !called {
		t.Fatalf("called = %v, error = %v", called, err)
	}
	got := authorizer.lastRequest()
	if got.Subject != "user:alice" || got.Action != "read" || got.Resource != (Resource{Type: "document", ID: "doc-1"}) {
		t.Fatalf("request = %#v", got)
	}
}

func TestServerSubjectOverride(t *testing.T) {
	authorizer := &fakeAuthorizer{allowed: true}
	_, err := invoke(t, Server(authorizer, rules(checkRule()), subject("workload:worker")), serverContext(testOperation), &wrapperspb.StringValue{Value: "doc-1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := authorizer.lastRequest().Subject; got != "workload:worker" {
		t.Fatalf("subject = %q", got)
	}
}

func TestServerMissingSubjectReturnsAuthn401(t *testing.T) {
	authorizer := &fakeAuthorizer{allowed: true}
	auditor := &captureAuditor{}
	_, err := invoke(t, Server(authorizer, rules(checkRule()), subject(""), WithAuditor(auditor)), serverContext(testOperation), &wrapperspb.StringValue{Value: "doc-1"}, nil)
	if !authnpb.IsAuthnErrorReasonUnauthenticated(err) {
		t.Fatalf("error = %v, want generated AuthN UNAUTHENTICATED", err)
	}
	if authorizer.requestCount() != 0 || len(auditor.all()) != 0 {
		t.Fatalf("authorizer calls = %d, audit events = %d", authorizer.requestCount(), len(auditor.all()))
	}
}

func TestServerDynamicResourceFailureReturns400(t *testing.T) {
	authorizer := &fakeAuthorizer{allowed: true}
	auditor := &captureAuditor{}
	_, err := invoke(t, Server(authorizer, rules(checkRule()), subject("user:alice"), WithAuditor(auditor)), serverContext(testOperation), &wrapperspb.StringValue{}, nil)
	if !authzpb.IsAuthzErrorReasonInvalidRequest(err) {
		t.Fatalf("error = %v, want generated INVALID_REQUEST", err)
	}
	if authorizer.requestCount() != 0 {
		t.Fatalf("authorizer calls = %d", authorizer.requestCount())
	}
	events := auditor.all()
	if len(events) != 1 {
		t.Fatalf("audit events = %d", len(events))
	}

	decision := decodeDecision(t, events[0])
	if decision.Reason != authzauditpb.AuthzDecision_REASON_INVALID_REQUEST || decision.Code != 400 {
		t.Fatalf("decision = %#v", decision)
	}
}
func TestServerDeniedReturns403(t *testing.T) {
	_, err := invoke(t, Server(&fakeAuthorizer{allowed: false}, rules(checkRule()), subject("user:alice")), serverContext(testOperation), &wrapperspb.StringValue{Value: "doc-1"}, nil)
	if !authzpb.IsAuthzErrorReasonDenied(err) {
		t.Fatalf("error = %v, want generated DENIED", err)
	}
}

func TestServerBackendErrorMappingAndWireSafety(t *testing.T) {
	providerCause := stderrors.New("provider endpoint token-secret detail")
	tests := []struct {
		name    string
		err     error
		matcher func(error) bool
		code    int32
		message string
	}{
		{name: "unavailable", err: stderrors.Join(ErrUnavailable, providerCause), matcher: authzpb.IsAuthzErrorReasonUnavailable, code: 503, message: "authorization service unavailable"},
		{name: "internal", err: providerCause, matcher: authzpb.IsAuthzErrorReasonInternal, code: 500, message: "internal authorization error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invoke(t, Server(&fakeAuthorizer{err: tt.err}, rules(checkRule()), subject("user:alice")), serverContext(testOperation), &wrapperspb.StringValue{Value: "doc-1"}, nil)
			if !tt.matcher(err) || !stderrors.Is(err, providerCause) {
				t.Fatalf("error = %v", err)
			}
			status := kerrors.FromError(err)
			if status.Code != tt.code || status.Message != tt.message {
				t.Fatalf("status = (%d, %q)", status.Code, status.Message)
			}
			grpcWire, marshalErr := json.Marshal(status.GRPCStatus().Proto())
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			httpWire, marshalErr := json.Marshal(status)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if bytes.Contains(grpcWire, []byte("token-secret")) || bytes.Contains(httpWire, []byte("token-secret")) {
				t.Fatalf("wire leaked cause: grpc=%s http=%s", grpcWire, httpWire)
			}
			if strings.Contains(err.Error(), "token-secret") {
				t.Fatalf("error string leaked cause: %v", err)
			}
		})
	}
}

func TestServerPreservesIncomingDeadline(t *testing.T) {
	var seen time.Time
	authorizer := &fakeAuthorizer{check: func(ctx context.Context, _ CheckRequest) (bool, error) {
		seen, _ = ctx.Deadline()
		return true, nil
	}}
	ctx, cancel := context.WithTimeout(serverContext(testOperation), time.Second)
	defer cancel()
	want, _ := ctx.Deadline()
	_, err := invoke(t, Server(authorizer, rules(checkRule()), subject("user:alice")), ctx, &wrapperspb.StringValue{Value: "doc-1"}, nil)
	if err != nil || !seen.Equal(want) {
		t.Fatalf("deadline = %v, want %v, error = %v", seen, want, err)
	}
}

func TestServerLoggerIsRootOnly(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))
	authorizer := &fakeAuthorizer{err: stderrors.New("backend token-secret")}
	_, err := invoke(t, Server(authorizer, rules(checkRule()), subject("identity-token-secret"), WithLogger(logger)), serverContext(testOperation), &wrapperspb.StringValue{Value: "resource-token-secret"}, nil)
	if err == nil {
		t.Fatal("error = nil")
	}
	if authorizer.setLoggers != 0 {
		t.Fatalf("backend logger injections = %d", authorizer.setLoggers)
	}
	logOutput := buffer.String()
	if !strings.Contains(logOutput, "REASON_INTERNAL") || strings.Contains(logOutput, "token-secret") {
		t.Fatalf("log = %q", logOutput)
	}
}

func TestServerTypedAuditOutcomes(t *testing.T) {
	providerCause := stderrors.New("provider-secret")
	tests := []struct {
		name     string
		allowed  bool
		err      error
		decision authzauditpb.AuthzDecision_Decision
		reason   authzauditpb.AuthzDecision_Reason
		code     int32
		event    string
	}{
		{name: "allowed", allowed: true, decision: authzauditpb.AuthzDecision_DECISION_ALLOWED, reason: authzauditpb.AuthzDecision_REASON_ALLOWED, code: 200, event: EventTypeAuthzAllowed},
		{name: "denied", decision: authzauditpb.AuthzDecision_DECISION_DENIED, reason: authzauditpb.AuthzDecision_REASON_DENIED, code: 403, event: EventTypeAuthzDenied},
		{name: "unavailable", err: stderrors.Join(ErrUnavailable, providerCause), decision: authzauditpb.AuthzDecision_DECISION_ERROR, reason: authzauditpb.AuthzDecision_REASON_UNAVAILABLE, code: 503, event: EventTypeAuthzError},
		{name: "internal", err: providerCause, decision: authzauditpb.AuthzDecision_DECISION_ERROR, reason: authzauditpb.AuthzDecision_REASON_INTERNAL, code: 500, event: EventTypeAuthzError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auditor := &captureAuditor{}
			_, _ = invoke(t, Server(&fakeAuthorizer{allowed: tt.allowed, err: tt.err}, rules(checkRule()), subject("user:alice"), WithAuditor(auditor)), serverContext(testOperation), &wrapperspb.StringValue{Value: "doc-1"}, nil)
			events := auditor.all()
			if len(events) != 1 {
				t.Fatalf("audit events = %d", len(events))
			}
			event := events[0]
			if event.Type() != tt.event || event.Subject() != "document:doc-1" {
				t.Fatalf("event = (%q, %q)", event.Type(), event.Subject())
			}
			if _, exists := event.Extensions()["authid"]; exists {
				t.Fatalf("authid extension exists: %v", event.Extensions())
			}
			decision := decodeDecision(t, event)
			if decision.Decision != tt.decision || decision.Reason != tt.reason || decision.Code != tt.code || decision.Subject != "user:alice" || decision.Action != "read" || decision.ResourceType != "document" || decision.ResourceId != "doc-1" {
				t.Fatalf("decision = %#v", decision)
			}
			if bytes.Contains(event.Data(), []byte("provider-secret")) {
				t.Fatalf("audit leaked provider cause: %q", event.Data())
			}
		})
	}
}

func TestServerAuditFailureDoesNotChangeDecision(t *testing.T) {
	auditor := &captureAuditor{err: stderrors.New("audit unavailable")}
	called := false
	_, err := invoke(t, Server(&fakeAuthorizer{allowed: true}, rules(checkRule()), subject("user:alice"), WithAuditor(auditor)), serverContext(testOperation), &wrapperspb.StringValue{Value: "doc-1"}, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})
	if err != nil || !called {
		t.Fatalf("called = %v, error = %v", called, err)
	}
}

func TestExtractProtoFieldNestedScalar(t *testing.T) {
	req := &descriptorpb.FileDescriptorProto{Options: &descriptorpb.FileOptions{GoPackage: proto.String("example.com/pkg")}}
	got, err := extractProtoField(req, "options.go_package")
	if err != nil || got != "example.com/pkg" {
		t.Fatalf("value = %q, error = %v", got, err)
	}
}

func decodeDecision(t *testing.T, event cloudevents.Event) *authzauditpb.AuthzDecision {
	t.Helper()
	decision := new(authzauditpb.AuthzDecision)
	if err := proto.Unmarshal(event.Data(), decision); err != nil {
		t.Fatal(err)
	}
	return decision
}
