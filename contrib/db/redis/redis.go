package redis

import (
	"context"
	stdtls "crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	redispb "github.com/Servora-Kit/servora/api/gen/go/servora/contrib/db/redis/v1"
	svrtls "github.com/Servora-Kit/servora/security/tls"
	"github.com/redis/go-redis/v9"
)

const (
	DefaultDialTimeout  = 5 * time.Second
	DefaultReadTimeout  = 3 * time.Second
	DefaultWriteTimeout = 3 * time.Second
)

type Client struct {
	rdb *redis.Client
	log *slog.Logger
}

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
func New(cfg *redispb.Redis, l *slog.Logger) (*Client, func(), error) {
	config, err := configFromProto(cfg)
	if err != nil {
		return nil, nil, err
	}
	return newClient(config, l)
}

func newClient(cfg *Config, l *slog.Logger) (*Client, func(), error) {
	if cfg == nil {
		return nil, nil, errors.New("redis config is nil")
	}
	if l == nil {
		l = slog.Default()
	}

	log := l.With("scope", "redis/contrib")
	rdb := redis.NewClient(newRedisOptions(cfg))

	ctx, cancel := context.WithTimeout(context.Background(), rdb.Options().DialTimeout)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		log.Error("redis ping failed", "err", err)
		return nil, nil, fmt.Errorf("redis ping: %w", err)
	}
	log.Info("redis client initialized")

	cleanup := func() {
		log.Info("closing redis connection")
		_ = rdb.Close()
	}

	return &Client{
		rdb: rdb,
		log: log,
	}, cleanup, nil
}

func newRedisOptions(cfg *Config) *redis.Options {
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
	return &redis.Options{
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

// Ping 测试连接
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Set 存储键值对
func (c *Client) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// Del 删除键
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// GetDel atomically gets the value and deletes the key (Redis >= 6.2).
func (c *Client) GetDel(ctx context.Context, key string) (string, error) {
	return c.rdb.GetDel(ctx, key).Result()
}

// Has 判断键是否存在
func (c *Client) Has(ctx context.Context, key string) bool {
	_, err := c.rdb.Get(ctx, key).Result()
	return err == nil
}

// Keys 按模式查找键
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.rdb.Keys(ctx, pattern).Result()
}

// SAdd 向集合添加成员
func (c *Client) SAdd(ctx context.Context, key string, members ...any) error {
	return c.rdb.SAdd(ctx, key, members...).Err()
}

// SMembers 获取集合所有成员
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.rdb.SMembers(ctx, key).Result()
}

// Expire 设置键过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.rdb.Expire(ctx, key, expiration).Err()
}
