package openfga

import (
	"bytes"
	"context"
	stderrors "errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Servora-Kit/servora/security/authz"
	fgasdk "github.com/openfga/go-sdk"
	fgaclient "github.com/openfga/go-sdk/client"
)

const (
	testStoreID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	testModelID = "01ARZ3NDEKTSV4RRFFQ69G5FAW"
)

func sdkClient(t *testing.T, handler http.HandlerFunc) (*fgaclient.OpenFgaClient, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	client, err := fgaclient.NewSdkClient(&fgaclient.ClientConfiguration{
		ApiUrl:               server.URL,
		StoreId:              testStoreID,
		AuthorizationModelId: testModelID,
		RetryParams:          &fgasdk.RetryParams{MaxRetry: 1, MinWaitInMs: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	return client, &calls
}

func request() authz.CheckRequest {
	return authz.CheckRequest{
		Subject:  "user:alice",
		Action:   "reader",
		Resource: authz.Resource{Type: "document", ID: "doc-1"},
	}
}

func TestNewValidation(t *testing.T) {
	if got, err := New(nil); err == nil || got != nil {
		t.Fatalf("authorizer = %v, error = %v", got, err)
	}
	client, _ := sdkClient(t, func(http.ResponseWriter, *http.Request) {})
	if got, err := New(client, nil); err == nil || got != nil || !strings.Contains(err.Error(), "option[0]") {
		t.Fatalf("authorizer = %v, error = %v", got, err)
	}
	if got, err := New(client, WithLogger(nil)); err == nil || got != nil || !strings.Contains(err.Error(), "option[0]") {
		t.Fatalf("authorizer = %v, error = %v", got, err)
	}
}

func TestDirectRequestValidationDoesNotCallSDK(t *testing.T) {
	client, calls := sdkClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("SDK called for invalid request")
	})
	authorizer, err := New(client)
	if err != nil {
		t.Fatal(err)
	}

	invalid := []authz.CheckRequest{
		{Action: "read", Resource: authz.Resource{Type: "document", ID: "1"}},
		{Subject: "user:1", Resource: authz.Resource{Type: "document", ID: "1"}},
		{Subject: "user:1", Action: "read", Resource: authz.Resource{ID: "1"}},
		{Subject: "user:1", Action: "read", Resource: authz.Resource{Type: "document"}},
	}
	for _, req := range invalid {
		if _, err := authorizer.Check(context.Background(), req); err == nil {
			t.Fatalf("Check(%#v) error = nil", req)
		}
	}
	if results, err := authorizer.BatchCheck(context.Background(), invalid[:1]); err == nil || results != nil {
		t.Fatalf("BatchCheck results = %v, error = %v", results, err)
	}
	if ids, err := authorizer.ListAllowed(context.Background(), "", "read", "document"); err == nil || ids != nil {
		t.Fatalf("ListAllowed ids = %v, error = %v", ids, err)
	}
	if calls.Load() != 0 {
		t.Fatalf("SDK calls = %d", calls.Load())
	}
}

func TestCheckMapsRequestAndAttributes(t *testing.T) {
	body := make(chan string, 1)
	client, _ := sdkClient(t, func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body <- string(data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"allowed":true}`))
	})
	authorizer, _ := New(client)
	req := request()
	req.Attributes = map[string]any{"tenant": "acme"}
	allowed, err := authorizer.Check(context.Background(), req)
	if err != nil || !allowed {
		t.Fatalf("allowed = %v, error = %v", allowed, err)
	}
	gotBody := <-body
	for _, want := range []string{`"user":"user:alice"`, `"relation":"reader"`, `"object":"document:doc-1"`, `"tenant":"acme"`} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("body missing %s: %s", want, gotBody)
		}
	}
}

func TestCheckProviderErrorClassification(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		unavailable bool
	}{
		{name: "rate limit", status: http.StatusTooManyRequests, unavailable: true},
		{name: "internal", status: http.StatusServiceUnavailable, unavailable: true},
		{name: "authentication", status: http.StatusUnauthorized},
		{name: "validation", status: http.StatusBadRequest},
		{name: "not found", status: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := sdkClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"code":"internal_error","message":"provider-secret"}`))
			})
			authorizer, _ := New(client)
			_, err := authorizer.Check(context.Background(), request())
			if err == nil || stderrors.Is(err, authz.ErrUnavailable) != tt.unavailable {
				t.Fatalf("error = %v, unavailable = %v", err, stderrors.Is(err, authz.ErrUnavailable))
			}
		})
	}
}

func TestCheckPreservesContextCancellation(t *testing.T) {
	client, _ := sdkClient(t, func(http.ResponseWriter, *http.Request) {})
	authorizer, _ := New(client)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := authorizer.Check(ctx, request())
	if !stderrors.Is(err, context.Canceled) || stderrors.Is(err, authz.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
}

func TestBatchCheckOrderAndCardinality(t *testing.T) {
	client, calls := sdkClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"1":{"allowed":false},"0":{"allowed":true}}}`))
	})
	authorizer, _ := New(client)
	if empty, err := authorizer.BatchCheck(context.Background(), nil); err != nil || len(empty) != 0 {
		t.Fatalf("empty = %v, error = %v", empty, err)
	}
	if calls.Load() != 0 {
		t.Fatalf("empty batch SDK calls = %d", calls.Load())
	}
	second := request()
	second.Resource.ID = "doc-2"
	results, err := authorizer.BatchCheck(context.Background(), []authz.CheckRequest{request(), second})
	if err != nil || len(results) != 2 || !results[0] || results[1] {
		t.Fatalf("results = %v, error = %v", results, err)
	}
}

func TestBatchCheckRejectsItemAndProtocolErrors(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		unavailable bool
	}{
		{name: "internal item", response: `{"result":{"0":{"error":{"internal_error":"unavailable","message":"secret"}}}}`, unavailable: true},
		{name: "input item", response: `{"result":{"0":{"error":{"input_error":"invalid_check_input","message":"secret"}}}}`},
		{name: "missing correlation", response: `{"result":{"unexpected":{"allowed":true}}}`},
		{name: "missing allowed", response: `{"result":{"0":{}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := sdkClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			})
			authorizer, _ := New(client)
			results, err := authorizer.BatchCheck(context.Background(), []authz.CheckRequest{request()})
			if err == nil || results != nil || stderrors.Is(err, authz.ErrUnavailable) != tt.unavailable {
				t.Fatalf("results = %v, error = %v", results, err)
			}
		})
	}
}

func TestListAllowedRequiresExactPrefix(t *testing.T) {
	tests := []struct {
		name     string
		objects  string
		want     []string
		wantFail bool
	}{
		{name: "valid", objects: `["document:1","document:2"]`, want: []string{"1", "2"}},
		{name: "wrong type", objects: `["folder:1"]`, wantFail: true},
		{name: "empty id", objects: `["document:"]`, wantFail: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := sdkClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"objects":` + tt.objects + `}`))
			})
			authorizer, _ := New(client)
			ids, err := authorizer.ListAllowed(context.Background(), "user:alice", "reader", "document")
			if tt.wantFail {
				if err == nil || ids != nil {
					t.Fatalf("ids = %v, error = %v", ids, err)
				}
				return
			}
			if err != nil || strings.Join(ids, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("ids = %v, error = %v", ids, err)
			}
		})
	}
}

func TestWithLoggerRecordsProviderFailure(t *testing.T) {
	var output bytes.Buffer
	client, _ := sdkClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"unauthenticated","message":"bad token"}`))
	})
	authorizer, err := New(client, WithLogger(slog.New(slog.NewTextHandler(&output, nil))))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = authorizer.Check(context.Background(), request())
	if !strings.Contains(output.String(), "operation=check") || !strings.Contains(output.String(), "reason=internal") {
		t.Fatalf("log = %q", output.String())
	}
	if strings.Contains(output.String(), "bad token") {
		t.Fatalf("log leaked provider response: %q", output.String())
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestCheckUnknownTransportFailureIsUnavailable(t *testing.T) {
	sentinel := stderrors.New("transport failed")
	client, err := fgaclient.NewSdkClient(&fgaclient.ClientConfiguration{
		ApiUrl:               "http://openfga.invalid",
		StoreId:              testStoreID,
		AuthorizationModelId: testModelID,
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, sentinel
		})},
		RetryParams: &fgasdk.RetryParams{MaxRetry: 1, MinWaitInMs: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	authorizer, _ := New(client)
	_, err = authorizer.Check(context.Background(), request())
	if !stderrors.Is(err, sentinel) || !stderrors.Is(err, authz.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckMalformedSuccessResponseRemainsInternal(t *testing.T) {
	client, _ := sdkClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{`))
	})
	authorizer, _ := New(client)
	_, err := authorizer.Check(context.Background(), request())
	if err == nil || stderrors.Is(err, authz.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckUnsupportedAttributesRemainInternal(t *testing.T) {
	client, calls := sdkClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("SDK sent an unsupported request")
	})
	authorizer, _ := New(client)
	req := request()
	req.Attributes = map[string]any{"unsupported": func() {}}
	_, err := authorizer.Check(context.Background(), req)
	if err == nil || stderrors.Is(err, authz.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("SDK calls = %d", calls.Load())
	}
}

func TestClassifyProviderErrorPrioritizesCanceledContext(t *testing.T) {
	providerError := stderrors.New("provider failed")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := classifyProviderError(ctx, "check", providerError)
	if !stderrors.Is(err, context.Canceled) || !stderrors.Is(err, providerError) || stderrors.Is(err, authz.ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
}
