package kafka

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	kafkapb "github.com/Servora-Kit/servora/api/gen/go/servora/contrib/kafka/v1"
	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestBuildOptsAppliesDefaultsAndAppendsExtra(t *testing.T) {
	cfg := &kafkapb.Kafka{
		Brokers:  []string{"127.0.0.1:9092"},
		ClientId: "servora-test",
	}

	opts, err := BuildOpts(cfg, WithSlogLogger(testLogger()), kgo.ClientID("override"))
	if err != nil {
		t.Fatalf("BuildOpts() error = %v", err)
	}
	if len(opts) == 0 {
		t.Fatal("BuildOpts() returned no options")
	}
	if cfg.GetRetryMax() != 3 {
		t.Fatalf("RetryMax default = %d, want 3", cfg.GetRetryMax())
	}
	if cfg.GetCompression() != "none" {
		t.Fatalf("Compression default = %q, want none", cfg.GetCompression())
	}
}

func TestBuildOptsLoggerOnlyWhenExplicit(t *testing.T) {
	cfg := &kafkapb.Kafka{Brokers: []string{"127.0.0.1:9092"}}
	withoutLogger, err := BuildOpts(cfg)
	if err != nil {
		t.Fatalf("BuildOpts() error = %v", err)
	}
	withLogger, err := BuildOpts(cfg, WithSlogLogger(testLogger()))
	if err != nil {
		t.Fatalf("BuildOpts(WithSlogLogger) error = %v", err)
	}
	if len(withLogger) != len(withoutLogger)+1 {
		t.Fatalf("option counts = (%d, %d), want explicit logger to add one option", len(withoutLogger), len(withLogger))
	}
}

func TestBuildOptsRejectsEmptyConfig(t *testing.T) {
	if _, err := BuildOpts(&kafkapb.Kafka{}); err == nil {
		t.Fatal("BuildOpts() error = nil, want required broker error")
	}
}

func TestBuildOptsRejectsUnsupportedSASL(t *testing.T) {
	cfg := &kafkapb.Kafka{
		Brokers: []string{"127.0.0.1:9092"},
		Sasl:    &kafkapb.Kafka_SASL{Mechanism: "UNKNOWN"},
	}
	if _, err := BuildOpts(cfg); err == nil {
		t.Fatal("BuildOpts() error = nil, want unsupported SASL error")
	}
}

func TestNewClientOptionalAbsent(t *testing.T) {
	client, err := NewClientOptional(context.Background(), nil)
	if err != nil {
		t.Fatalf("NewClientOptional(nil) error = %v", err)
	}
	if client != nil {
		t.Fatal("NewClientOptional(nil) returned non-nil client")
	}

	client, err = NewClientOptional(context.Background(), &kafkapb.Kafka{})
	if err != nil {
		t.Fatalf("NewClientOptional(empty) error = %v", err)
	}
	if client != nil {
		t.Fatal("NewClientOptional(empty) returned non-nil client")
	}
}

func TestNewClientOptionalReturnsPingError(t *testing.T) {
	cfg := &kafkapb.Kafka{
		Brokers:     []string{"127.0.0.1:1"},
		DialTimeout: durationpb.New(time.Millisecond),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	client, err := NewClientOptional(ctx, cfg)
	if err == nil {
		if client != nil {
			client.Close()
		}
		t.Fatal("NewClientOptional() error = nil, want ping error")
	}
	if client != nil {
		t.Fatal("NewClientOptional() returned client with ping error")
	}
}

func TestWithSlogLoggerExplicitOption(t *testing.T) {
	if WithSlogLogger(testLogger()) == nil {
		t.Fatal("WithSlogLogger() returned nil option")
	}
	if WithSlogLogger(nil) == nil {
		t.Fatal("WithSlogLogger(nil) returned nil option")
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
