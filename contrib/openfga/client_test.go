package openfga

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	openfgaconfpb "github.com/Servora-Kit/servora/api/gen/go/servora/contrib/openfga/v1"
	fgaclient "github.com/openfga/go-sdk/client"
	"google.golang.org/protobuf/proto"
)

const (
	testStoreID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	testModelID = "01ARZ3NDEKTSV4RRFFQ69G5FAW"
)

type capturedRequest struct {
	authorization string
	path          string
	body          string
}

func TestNewClientValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  *openfgaconfpb.OpenFGA
	}{
		{name: "nil"},
		{name: "missing api url", cfg: &openfgaconfpb.OpenFGA{StoreId: testStoreID}},
		{name: "missing store", cfg: &openfgaconfpb.OpenFGA{ApiUrl: "http://127.0.0.1"}},
		{name: "api token over plaintext HTTP", cfg: &openfgaconfpb.OpenFGA{ApiUrl: "http://openfga.example", StoreId: testStoreID, ApiToken: "secret-token"}},
		{name: "API token without hostname", cfg: &openfgaconfpb.OpenFGA{ApiUrl: "https://:443", StoreId: testStoreID, ApiToken: "secret-token"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if client, err := NewClient(tt.cfg); err == nil || client != nil {
				t.Fatalf("client = %v, error = %v", client, err)
			}
		})
	}
}

func TestNewClientMapsConfigWithoutMutationOrNetwork(t *testing.T) {
	var requests atomic.Int32
	captured := make(chan capturedRequest, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		data, _ := io.ReadAll(r.Body)
		captured <- capturedRequest{
			authorization: r.Header.Get("Authorization"),
			path:          r.URL.Path,
			body:          string(data),
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	defer server.Close()
	previousDefaultClient := http.DefaultClient
	http.DefaultClient = server.Client()
	defer func() { http.DefaultClient = previousDefaultClient }()

	cfg := &openfgaconfpb.OpenFGA{
		ApiUrl:   server.URL,
		StoreId:  testStoreID,
		ModelId:  testModelID,
		ApiToken: "secret-token",
	}
	before := proto.Clone(cfg)
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 0 {
		t.Fatalf("constructor requests = %d", requests.Load())
	}
	if !proto.Equal(cfg, before) {
		t.Fatalf("config mutated: before=%v after=%v", before, cfg)
	}

	response, err := client.Check(context.Background()).Body(fgaclient.ClientCheckRequest{
		User:     "user:alice",
		Relation: "reader",
		Object:   "document:doc-1",
	}).Execute()
	if err != nil || !response.GetAllowed() {
		t.Fatalf("allowed = %v, error = %v", response.GetAllowed(), err)
	}
	request := <-captured
	if request.authorization != "Bearer secret-token" {
		t.Fatalf("authorization = %q", request.authorization)
	}
	if !strings.Contains(request.path, "/stores/"+testStoreID+"/check") {
		t.Fatalf("path = %q", request.path)
	}
	if !strings.Contains(request.body, testModelID) {
		t.Fatalf("body missing model id: %s", request.body)
	}
}

func TestNewClientRejectsRedirectWhenTokenConfigured(t *testing.T) {
	var plaintextHits atomic.Int32
	plaintext := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		plaintextHits.Add(1)
	}))
	defer plaintext.Close()

	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, plaintext.URL, http.StatusTemporaryRedirect)
	}))
	defer secure.Close()

	previousDefaultClient := http.DefaultClient
	http.DefaultClient = secure.Client()
	defer func() { http.DefaultClient = previousDefaultClient }()

	client, err := NewClient(&openfgaconfpb.OpenFGA{
		ApiUrl:   secure.URL,
		StoreId:  testStoreID,
		ApiToken: "secret-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Check(context.Background()).Body(fgaclient.ClientCheckRequest{
		User: "user:alice", Relation: "reader", Object: "document:doc-1",
	}).Execute()
	if err == nil {
		t.Fatal("redirected request error = nil")
	}
	if plaintextHits.Load() != 0 {
		t.Fatalf("plaintext endpoint hits = %d", plaintextHits.Load())
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("redirect error leaked token: %v", err)
	}
}

func TestNewClientOpenFGAIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires OpenFGA")
	}
	apiURL := os.Getenv("OPENFGA_TEST_API_URL")
	if apiURL == "" {
		apiURL = startOpenFGACompose(t)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bootstrap, err := fgaclient.NewSdkClient(&fgaclient.ClientConfiguration{ApiUrl: apiURL})
	if err != nil {
		t.Fatal(err)
	}
	store := createStoreEventually(t, ctx, bootstrap)
	client, err := NewClient(&openfgaconfpb.OpenFGA{ApiUrl: apiURL, StoreId: store.GetId()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := client.DeleteStore(cleanupCtx).Execute(); err != nil {
			t.Errorf("delete integration store: %v", err)
		}
	})
	if _, err := client.Read(ctx).Body(fgaclient.ClientReadRequest{}).Execute(); err != nil {
		t.Fatalf("Read() through NewClient: %v", err)
	}
}

func createStoreEventually(t *testing.T, ctx context.Context, client *fgaclient.OpenFgaClient) *fgaclient.ClientCreateStoreResponse {
	t.Helper()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		store, err := client.CreateStore(ctx).Body(fgaclient.ClientCreateStoreRequest{Name: "servora-client-integration"}).Execute()
		if err == nil {
			return store
		}
		select {
		case <-ctx.Done():
			t.Fatalf("wait for OpenFGA: %v", err)
			return nil
		case <-ticker.C:
		}
	}
}

func startOpenFGACompose(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve OpenFGA integration test path")
	}
	composeFile := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "docker-compose.yaml"))
	project := fmt.Sprintf("servora-openfga-test-%d", os.Getpid())

	startCtx, startCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer startCancel()
	start := openFGAComposeCommand(startCtx, composeFile, project, "up", "-d", "--wait", "postgres", "openfga")
	if output, err := start.CombinedOutput(); err != nil {
		cleanupOpenFGACompose(t, composeFile, project)
		t.Fatalf("start OpenFGA compose: %v\n%s", err, output)
	}
	t.Cleanup(func() { cleanupOpenFGACompose(t, composeFile, project) })

	portCtx, portCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer portCancel()
	port := openFGAComposeCommand(portCtx, composeFile, project, "port", "openfga", "8080")
	output, err := port.Output()
	if err != nil {
		t.Fatalf("resolve OpenFGA compose port: %v", err)
	}
	host, assignedPort, err := net.SplitHostPort(strings.TrimSpace(string(output)))
	if err != nil {
		t.Fatalf("parse OpenFGA compose port %q: %v", output, err)
	}
	return "http://" + net.JoinHostPort(host, assignedPort)
}

func openFGAComposeCommand(ctx context.Context, composeFile, project string, args ...string) *exec.Cmd {
	base := []string{"compose", "--project-name", project, "--file", composeFile}
	command := exec.CommandContext(ctx, "docker", append(base, args...)...)
	command.Env = openFGAComposeEnv()
	return command
}

func openFGAComposeEnv() []string {
	blocked := map[string]struct{}{
		"POSTGRES_PORT":           {},
		"OPENFGA_HTTP_PORT":       {},
		"OPENFGA_GRPC_PORT":       {},
		"OPENFGA_PLAYGROUND_PORT": {},
	}
	environment := make([]string, 0, len(os.Environ())+4)
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		if _, found := blocked[name]; !found {
			environment = append(environment, value)
		}
	}
	return append(environment,
		"POSTGRES_PORT=0",
		"OPENFGA_HTTP_PORT=0",
		"OPENFGA_GRPC_PORT=0",
		"OPENFGA_PLAYGROUND_PORT=0",
	)
}

func cleanupOpenFGACompose(t *testing.T, composeFile, project string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := openFGAComposeCommand(ctx, composeFile, project, "down", "--remove-orphans", "--volumes")
	if output, err := command.CombinedOutput(); err != nil {
		t.Errorf("stop OpenFGA compose: %v\n%s", err, output)
	}
}

func TestNewClientEmptyModelRemainsUnset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("constructor performed a network request")
	}))
	defer server.Close()

	client, err := NewClient(&openfgaconfpb.OpenFGA{ApiUrl: server.URL, StoreId: testStoreID})
	if err != nil {
		t.Fatal(err)
	}
	modelID, err := client.GetAuthorizationModelId()
	if err != nil || modelID != "" {
		t.Fatalf("model id = %q, error = %v", modelID, err)
	}
}

func TestNewClientErrorDoesNotLeakToken(t *testing.T) {
	_, err := NewClient(&openfgaconfpb.OpenFGA{
		ApiUrl:   "://invalid",
		StoreId:  testStoreID,
		ApiToken: "secret-token",
	})
	if err == nil {
		t.Fatal("error = nil")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error leaked token: %v", err)
	}
}
