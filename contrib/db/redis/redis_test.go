package redis

import (
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	redispb "github.com/Servora-Kit/servora/api/gen/go/servora/contrib/db/redis/v1"
	tlspb "github.com/Servora-Kit/servora/api/gen/go/servora/security/tls/v1"
)

func TestConfigFromProtoPlaintext(t *testing.T) {
	t.Parallel()

	cfg, err := configFromProto(&redispb.Redis{Addr: "localhost:6379"})
	if err != nil {
		t.Fatalf("configFromProto() error = %v", err)
	}
	if cfg.TLSConfig != nil {
		t.Fatalf("TLSConfig = %#v, want nil", cfg.TLSConfig)
	}
	if cfg.DialTimeout != DefaultDialTimeout || cfg.ReadTimeout != DefaultReadTimeout || cfg.WriteTimeout != DefaultWriteTimeout {
		t.Fatalf("timeouts = (%v, %v, %v), want defaults", cfg.DialTimeout, cfg.ReadTimeout, cfg.WriteTimeout)
	}
}

func TestConfigFromProtoTLSUsesAddrHost(t *testing.T) {
	t.Parallel()

	cfg, err := configFromProto(&redispb.Redis{
		Addr: "redis.internal:6380",
		Tls:  &tlspb.TLS{Enable: true},
	})
	if err != nil {
		t.Fatalf("configFromProto() error = %v", err)
	}
	if cfg.TLSConfig == nil {
		t.Fatal("TLSConfig = nil, want enabled TLS")
	}
	if cfg.TLSConfig.ServerName != "redis.internal" {
		t.Fatalf("ServerName = %q, want redis.internal", cfg.TLSConfig.ServerName)
	}
	if cfg.TLSConfig.InsecureSkipVerify {
		t.Fatal("TLSConfig unexpectedly disables certificate verification")
	}
	if got := newRedisOptions(cfg).TLSConfig; got != cfg.TLSConfig {
		t.Fatal("go-redis options did not receive the configured TLSConfig")
	}
}

func TestConfigFromProtoTLSRejectsAddrWithoutPort(t *testing.T) {
	t.Parallel()

	_, err := configFromProto(&redispb.Redis{
		Addr: "redis.internal",
		Tls:  &tlspb.TLS{Enable: true},
	})
	if err == nil || !strings.Contains(err.Error(), "host:port") {
		t.Fatalf("configFromProto() error = %v, want host:port error", err)
	}
}

func TestNewClientReportsConnectionFailure(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client, cleanup, err := newClient(&Config{
		Addr:         "127.0.0.1:1",
		DialTimeout:  20 * time.Millisecond,
		ReadTimeout:  20 * time.Millisecond,
		WriteTimeout: 20 * time.Millisecond,
	}, logger)
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		t.Fatal("NewClient() error = nil, want connection failure")
	}
	if client != nil || cleanup != nil {
		t.Fatalf("NewClient() returned client=%t cleanup=%t, want both nil", client != nil, cleanup != nil)
	}
	if !strings.Contains(err.Error(), "redis ping") {
		t.Fatalf("NewClient() error = %v, want redis ping context", err)
	}
}
