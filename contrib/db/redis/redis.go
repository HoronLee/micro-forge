package redis

import (
	"context"
	stdtls "crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	redispb "github.com/Servora-Kit/servora/api/gen/go/servora/contrib/db/redis/v1"
	svrtls "github.com/Servora-Kit/servora/security/tls"
	goredis "github.com/redis/go-redis/v9"
)

const (
	DefaultDialTimeout  = 5 * time.Second
	DefaultReadTimeout  = 3 * time.Second
	DefaultWriteTimeout = 3 * time.Second
)

type Config struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	TLSConfig    *stdtls.Config
}

func configFromProto(cfg *redispb.Redis) (*Config, error) {
	if cfg == nil {
		return nil, nil
	}

	config := &Config{
		Addr:     cfg.GetAddr(),
		Username: cfg.GetUserName(),
		Password: cfg.GetPassword(),
		DB:       int(cfg.GetDb()),
	}

	if cfg.GetDialTimeout() != nil {
		config.DialTimeout = cfg.GetDialTimeout().AsDuration()
	} else {
		config.DialTimeout = DefaultDialTimeout
	}

	if cfg.GetReadTimeout() != nil {
		config.ReadTimeout = cfg.GetReadTimeout().AsDuration()
	} else {
		config.ReadTimeout = DefaultReadTimeout
	}

	if cfg.GetWriteTimeout() != nil {
		config.WriteTimeout = cfg.GetWriteTimeout().AsDuration()
	} else {
		config.WriteTimeout = DefaultWriteTimeout
	}

	var serverName string
	if cfg.GetTls().GetEnable() {
		host, _, err := net.SplitHostPort(cfg.GetAddr())
		if err != nil || host == "" {
			return nil, fmt.Errorf("redis TLS requires addr in host:port form: %q", cfg.GetAddr())
		}
		serverName = host
	}
	tlsConfig, err := svrtls.BuildClientTLSForServer(cfg.GetTls(), serverName)
	if err != nil {
		return nil, fmt.Errorf("build redis TLS config: %w", err)
	}
	config.TLSConfig = tlsConfig
	return config, nil
}

// New creates a Redis client from the shared Redis configuration, verifies the connection, and returns cleanup.
func New(cfg *redispb.Redis) (*goredis.Client, func(), error) {
	config, err := configFromProto(cfg)
	if err != nil {
		return nil, nil, err
	}
	return newClient(config)
}

func newClient(cfg *Config) (*goredis.Client, func(), error) {
	if cfg == nil {
		return nil, nil, errors.New("redis config is nil")
	}

	rdb := goredis.NewClient(newRedisOptions(cfg))

	ctx, cancel := context.WithTimeout(context.Background(), rdb.Options().DialTimeout)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, nil, fmt.Errorf("redis ping: %w", err)
	}

	cleanup := func() {
		_ = rdb.Close()
	}

	return rdb, cleanup, nil
}

func newRedisOptions(cfg *Config) *goredis.Options {
	dialTimeout := cfg.DialTimeout
	if dialTimeout == 0 {
		dialTimeout = DefaultDialTimeout
	}
	readTimeout := cfg.ReadTimeout
	if readTimeout == 0 {
		readTimeout = DefaultReadTimeout
	}
	writeTimeout := cfg.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = DefaultWriteTimeout
	}
	return &goredis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		TLSConfig:    cfg.TLSConfig,
	}
}
