package data

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/Servora-Kit/servora/pkg/logger"
	"github.com/Servora-Kit/servora/pkg/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRedisAddr() string {
	if addr := os.Getenv("REDIS_TEST_ADDR"); addr != "" {
		return addr
	}
	return "127.0.0.1:6379"
}

func testRedisPassword() string {
	return os.Getenv("REDIS_TEST_PASSWORD")
}

func testRedisDB(defaultDB int) int {
	if db := os.Getenv("REDIS_TEST_DB"); db != "" {
		if parsed, err := strconv.Atoi(db); err == nil {
			return parsed
		}
	}
	return defaultDB
}

// setupTestAuthRepo 设置测试用的AuthRepo
func setupTestAuthRepo(t *testing.T) (*authRepo, func()) {
	// 设置Redis客户端
	cfg := &redis.Config{
		Addr:         testRedisAddr(),
		Password:     testRedisPassword(),
		DB:           testRedisDB(2), // 使用专门的测试数据库
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	testLogger := logger.NewLogger(&logger.Config{
		Env:   "test",
		Level: 1, // info level
	})

	redisClient, redisCleanup, err := redis.NewClient(cfg, testLogger)
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}

	// 清理测试数据库
	ctx := context.Background()
	cleanupByPattern := func(pattern string) {
		keys, err := redisClient.Keys(ctx, pattern)
		if err != nil {
			t.Logf("failed to list redis keys by pattern %q: %v", pattern, err)
			return
		}
		if len(keys) == 0 {
			return
		}
		if err := redisClient.Del(ctx, keys...); err != nil {
			t.Logf("failed to delete redis keys by pattern %q: %v", pattern, err)
		}
	}

	cleanupByPattern("refresh_token:*")
	cleanupByPattern("user_tokens:*")

	// 创建Data结构体
	data := &Data{
		redis: redisClient,
	}

	// 创建authRepo
	repo := &authRepo{
		data: data,
		log:  log.NewHelper(testLogger),
	}

	cleanup := func() {
		// 清理测试数据
		cleanupByPattern("refresh_token:*")
		cleanupByPattern("user_tokens:*")
		redisCleanup()
	}

	return repo, cleanup
}

// generateRandomToken 生成随机token字符串
func generateRandomToken() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// generateRandomUserID 生成随机用户ID
func generateRandomUserID() int64 {
	return rand.Int63n(10000) + 1 // 1-10000之间的随机数
}

// TestProperty_RefreshTokenStorageConsistency 属性测试：Refresh Token存储一致性
// **属性2: Refresh Token存储一致性**
// **验证: 需求 3.3, 3.4**
//
// 对于任意成功登录生成的Refresh Token，该Token应存在于Redis中，且关联的用户ID应与登录用户一致。
func TestProperty_RefreshTokenStorageConsistency(t *testing.T) {
	repo, cleanup := setupTestAuthRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 运行100次随机测试
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// 生成随机测试数据
			userID := generateRandomUserID()
			token := generateRandomToken()
			expiration := time.Duration(rand.Intn(3600)+60) * time.Second // 1分钟到1小时

			// 保存Refresh Token
			err := repo.SaveRefreshToken(ctx, userID, token, expiration)
			require.NoError(t, err, "SaveRefreshToken should succeed")

			// 验证Token存在于Redis中
			retrievedUserID, err := repo.GetRefreshToken(ctx, token)
			require.NoError(t, err, "GetRefreshToken should succeed for saved token")

			// 验证关联的用户ID一致
			assert.Equal(t, userID, retrievedUserID,
				"Retrieved user ID should match the original user ID")

			// 验证Token在用户Token集合中存在
			userTokensKey := fmt.Sprintf("user_tokens:%d", userID)
			tokens, err := repo.data.redis.SMembers(ctx, userTokensKey)
			require.NoError(t, err, "Should be able to get user tokens")
			assert.Contains(t, tokens, token, "Token should be in user tokens set")
			assert.Greater(t, len(tokens), 0, "User tokens set should not be empty")

			// 验证Token键的格式正确
			tokenKey := fmt.Sprintf("refresh_token:%s", token)
			storedValue, err := repo.data.redis.Get(ctx, tokenKey)
			require.NoError(t, err, "Token key should exist in Redis")
			assert.Equal(t, strconv.FormatInt(userID, 10), storedValue,
				"Stored value should be the user ID as string")

			// 清理这次测试的数据
			err = repo.DeleteRefreshToken(ctx, token)
			require.NoError(t, err, "Should be able to clean up token")
		})
	}
}

// TestProperty_RefreshTokenStorageConsistency_MultipleTokens 测试同一用户多个Token的存储一致性
func TestProperty_RefreshTokenStorageConsistency_MultipleTokens(t *testing.T) {
	repo, cleanup := setupTestAuthRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 运行50次随机测试，每次为同一用户创建多个Token
	for i := 0; i < 50; i++ {
		t.Run(fmt.Sprintf("multi_token_iteration_%d", i), func(t *testing.T) {
			userID := generateRandomUserID()
			tokenCount := rand.Intn(5) + 2 // 2-6个Token
			tokens := make([]string, tokenCount)
			expiration := time.Duration(rand.Intn(3600)+60) * time.Second

			// 为同一用户创建多个Token
			for j := 0; j < tokenCount; j++ {
				tokens[j] = generateRandomToken()
				err := repo.SaveRefreshToken(ctx, userID, tokens[j], expiration)
				require.NoError(t, err, "SaveRefreshToken should succeed for token %d", j)
			}

			// 验证所有Token都能正确检索
			for j, token := range tokens {
				retrievedUserID, err := repo.GetRefreshToken(ctx, token)
				require.NoError(t, err, "GetRefreshToken should succeed for token %d", j)
				assert.Equal(t, userID, retrievedUserID,
					"Retrieved user ID should match for token %d", j)
			}

			// 验证用户Token集合包含所有Token
			userTokensKey := fmt.Sprintf("user_tokens:%d", userID)
			storedTokens, err := repo.data.redis.SMembers(ctx, userTokensKey)
			require.NoError(t, err, "Should be able to get user tokens")
			assert.Len(t, storedTokens, tokenCount, "User tokens set should contain all tokens")

			for j, token := range tokens {
				assert.Contains(t, storedTokens, token, "Token %d should be in user tokens set", j)
			}

			// 清理测试数据
			err = repo.DeleteUserRefreshTokens(ctx, userID)
			require.NoError(t, err, "Should be able to clean up all user tokens")
		})
	}
}

// TestProperty_RefreshTokenStorageConsistency_EdgeCases 测试边界情况的存储一致性
func TestProperty_RefreshTokenStorageConsistency_EdgeCases(t *testing.T) {
	repo, cleanup := setupTestAuthRepo(t)
	defer cleanup()

	ctx := context.Background()

	testCases := []struct {
		name       string
		userID     int64
		token      string
		expiration time.Duration
	}{
		{
			name:       "minimum_user_id",
			userID:     1,
			token:      generateRandomToken(),
			expiration: time.Minute,
		},
		{
			name:       "maximum_user_id",
			userID:     9223372036854775807, // max int64
			token:      generateRandomToken(),
			expiration: time.Hour * 24,
		},
		{
			name:       "short_token",
			userID:     generateRandomUserID(),
			token:      "abc",
			expiration: time.Hour,
		},
		{
			name:       "long_token",
			userID:     generateRandomUserID(),
			token:      generateRandomToken() + generateRandomToken() + generateRandomToken(),
			expiration: time.Hour,
		},
		{
			name:       "minimum_expiration",
			userID:     generateRandomUserID(),
			token:      generateRandomToken(),
			expiration: time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 保存Token
			err := repo.SaveRefreshToken(ctx, tc.userID, tc.token, tc.expiration)
			require.NoError(t, err, "SaveRefreshToken should succeed")

			// 验证存储一致性
			retrievedUserID, err := repo.GetRefreshToken(ctx, tc.token)
			require.NoError(t, err, "GetRefreshToken should succeed")
			assert.Equal(t, tc.userID, retrievedUserID, "User ID should match")

			// 验证在用户Token集合中
			userTokensKey := fmt.Sprintf("user_tokens:%d", tc.userID)
			tokens, err := repo.data.redis.SMembers(ctx, userTokensKey)
			require.NoError(t, err, "Should be able to get user tokens")
			assert.Contains(t, tokens, tc.token, "Token should be in user tokens set")

			// 清理
			err = repo.DeleteRefreshToken(ctx, tc.token)
			require.NoError(t, err, "Should be able to clean up")
		})
	}
}
