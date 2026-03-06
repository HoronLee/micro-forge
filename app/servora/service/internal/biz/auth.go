package biz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	authpb "github.com/Servora-Kit/servora/api/gen/go/auth/service/v1"
	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
	"github.com/Servora-Kit/servora/app/servora/service/internal/biz/entity"
	dataent "github.com/Servora-Kit/servora/app/servora/service/internal/data/ent"
	"github.com/Servora-Kit/servora/pkg/helpers"
	jwtpkg "github.com/Servora-Kit/servora/pkg/jwt"
	"github.com/Servora-Kit/servora/pkg/logger"
)

// AuthUsecase is a Auth usecase.
type AuthUsecase struct {
	repo            AuthRepo
	log             *logger.Helper
	cfg             *conf.App
	adminRegistered bool                    // 是否已经注册了 admin 用户
	accessJWT       *jwtpkg.JWT[UserClaims] // Access Token JWT service
	refreshJWT      *jwtpkg.JWT[UserClaims] // Refresh Token JWT service (for validation only)
}

// NewAuthUsecase new an auth usecase.
func NewAuthUsecase(repo AuthRepo, l logger.Logger, cfg *conf.App) *AuthUsecase {
	accessJWTService := jwtpkg.NewJWT[UserClaims](&jwtpkg.Config{
		SecretKey: cfg.Jwt.AccessSecret,
	})

	refreshJWTService := jwtpkg.NewJWT[UserClaims](&jwtpkg.Config{
		SecretKey: cfg.Jwt.RefreshSecret,
	})

	uc := &AuthUsecase{
		repo:       repo,
		log:        logger.NewHelper(l, logger.WithModule("auth/biz/servora-service")),
		cfg:        cfg,
		accessJWT:  accessJWTService,
		refreshJWT: refreshJWTService,
	}
	admin, err := repo.GetUserByUserName(context.Background(), "admin")
	if err == nil && admin != nil {
		uc.adminRegistered = true
	}
	return uc
}

// UserClaims defines the custom claims for the JWT.
// It embeds jwt.RegisteredClaims to include standard JWT fields.
type UserClaims struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	Nonce string `json:"nonce"` // Random nonce to ensure token uniqueness
	jwt.RegisteredClaims
}

// TokenPair represents a pair of access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // Access token expiration time in seconds
}

// TokenStore Refresh Token存储接口
type TokenStore interface {
	// SaveRefreshToken 保存Refresh Token
	SaveRefreshToken(ctx context.Context, userID int64, token string, expiration time.Duration) error

	// GetRefreshToken 获取Refresh Token关联的用户ID
	GetRefreshToken(ctx context.Context, token string) (int64, error)

	// DeleteRefreshToken 删除Refresh Token
	DeleteRefreshToken(ctx context.Context, token string) error

	// DeleteUserRefreshTokens 删除用户所有Refresh Token
	DeleteUserRefreshTokens(ctx context.Context, userID int64) error
}

// AuthRepo 统一的认证仓库接口，包含数据库和 grpc 操作
type AuthRepo interface {
	// 数据库操作
	SaveUser(context.Context, *entity.User) (*entity.User, error)
	GetUserByEmail(context.Context, string) (*entity.User, error)
	GetUserByUserName(context.Context, string) (*entity.User, error)
	GetUserByID(context.Context, int64) (*entity.User, error)
	// Token存储方法
	TokenStore
}

// SignupByEmail 使用邮件注册
func (uc *AuthUsecase) SignupByEmail(ctx context.Context, user *entity.User) (*entity.User, error) {
	// 检查 admin 用户是否已存在
	if !uc.adminRegistered {
		// 第一次注册，用户名必须为 admin
		if user.Name != "admin" {
			return nil, authpb.ErrorInvalidCredentials("the first user must be named admin")
		}
		user.Role = "admin"
	} else {
		// 后续注册，用户名可以任意，但角色为 user
		// 检查用户名是否已存在
		existingUser, err := uc.repo.GetUserByUserName(ctx, user.Name)
		if err != nil && !dataent.IsNotFound(err) {
			return nil, authpb.ErrorUserNotFound("failed to check username: %v", err)
		}
		if existingUser != nil {
			return nil, authpb.ErrorUserAlreadyExists("username already exists")
		}
		user.Role = "user"
	}

	// 检查邮箱是否已存在
	existingEmail, err := uc.repo.GetUserByEmail(ctx, user.Email)
	if err != nil && !dataent.IsNotFound(err) {
		return nil, authpb.ErrorUserNotFound("failed to check email: %v", err)
	}
	if existingEmail != nil {
		return nil, authpb.ErrorUserAlreadyExists("email already exists")
	}

	createdUser, err := uc.repo.SaveUser(ctx, user)
	if err == nil && !uc.adminRegistered && user.Name == "admin" {
		uc.adminRegistered = true // 注册成功后更新状态
	}
	return createdUser, err
}

// generateAccessToken 签发 Access Token
func (uc *AuthUsecase) generateAccessToken(claims *UserClaims) (string, error) {
	return uc.accessJWT.GenerateToken(claims)
}

// generateRefreshToken 生成 Refresh Token (UUID-like string)
func (uc *AuthUsecase) generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// LoginByEmailPassword 邮箱密码登录 - 返回Token Pair
func (uc *AuthUsecase) LoginByEmailPassword(ctx context.Context, user *entity.User) (*TokenPair, error) {
	foundUser, err := uc.repo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return nil, authpb.ErrorUserNotFound("failed to get user: %v", err)
	}
	if foundUser == nil {
		uc.log.Warnf("user %s does not exist", user.Email)
		return nil, authpb.ErrorUserNotFound("user %s does not exist", user.Email)
	}
	if !helpers.BcryptCheck(user.Password, foundUser.Password) {
		return nil, authpb.ErrorIncorrectPassword("incorrect password for user: %s", user.Email)
	}

	// 登录成功，生成Token Pair

	// Generate a random nonce to ensure token uniqueness
	nonce, err := uc.generateRefreshToken()
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate nonce: %v", err)
	}

	// 生成Access Token
	accessClaims := &UserClaims{
		ID:    foundUser.ID,
		Name:  foundUser.Name,
		Role:  foundUser.Role,
		Nonce: nonce,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{uc.cfg.Jwt.Audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(uc.cfg.Jwt.AccessExpire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    uc.cfg.Jwt.Issuer,
		},
	}

	accessToken, err := uc.generateAccessToken(accessClaims)
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate access token: %v", err)
	}

	// 生成Refresh Token
	refreshToken, err := uc.generateRefreshToken()
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate refresh token: %v", err)
	}

	// 保存Refresh Token到Redis
	refreshExpirationTime := time.Duration(uc.cfg.Jwt.RefreshExpire) * time.Second
	if err := uc.repo.SaveRefreshToken(ctx, foundUser.ID, refreshToken, refreshExpirationTime); err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to save refresh token: %v", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(uc.cfg.Jwt.AccessExpire),
	}, nil
}

// RefreshToken 刷新Access Token
func (uc *AuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// 从Redis获取Refresh Token关联的用户ID
	userID, err := uc.repo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		uc.log.Warnf("Invalid refresh token: %v", err)
		return nil, authpb.ErrorInvalidRefreshToken("invalid or expired refresh token")
	}

	// 获取用户信息
	user, err := uc.repo.GetUserByID(ctx, userID)
	if err != nil {
		uc.log.Errorf("Failed to get user by ID: %v", err)
		return nil, authpb.ErrorUserNotFound("user not found: %v", err)
	}

	// 生成新的Access Token
	accessExpirationTime := time.Duration(uc.cfg.Jwt.AccessExpire) * time.Second

	// Generate a random nonce to ensure token uniqueness
	nonce, err := uc.generateRefreshToken() // Reuse the random generation logic
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate nonce: %v", err)
	}

	accessClaims := &UserClaims{
		ID:    user.ID,
		Name:  user.Name,
		Role:  user.Role,
		Nonce: nonce,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{uc.cfg.Jwt.Audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessExpirationTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    uc.cfg.Jwt.Issuer,
		},
	}

	accessToken, err := uc.generateAccessToken(accessClaims)
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate access token: %v", err)
	}

	// 可选：轮换Refresh Token
	// 这里我们生成新的Refresh Token并删除旧的
	newRefreshToken, err := uc.generateRefreshToken()
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate refresh token: %v", err)
	}

	// 删除旧的Refresh Token
	if err := uc.repo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		uc.log.Warnf("Failed to delete old refresh token: %v", err)
		// 不返回错误，继续执行
	}

	// 保存新的Refresh Token
	refreshExpirationTime := time.Duration(uc.cfg.Jwt.RefreshExpire) * time.Second
	if err := uc.repo.SaveRefreshToken(ctx, user.ID, newRefreshToken, refreshExpirationTime); err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to save refresh token: %v", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(uc.cfg.Jwt.AccessExpire),
	}, nil
}

// Logout 登出，使Refresh Token失效
func (uc *AuthUsecase) Logout(ctx context.Context, refreshToken string) error {
	// 删除Refresh Token
	if err := uc.repo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		uc.log.Warnf("Failed to delete refresh token during logout: %v", err)
		// 即使删除失败，也返回成功，因为token可能已经不存在
	}
	return nil
}
