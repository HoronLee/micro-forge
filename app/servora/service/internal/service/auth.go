package service

import (
	"context"
	"fmt"

	authpb "github.com/Servora-Kit/servora/api/gen/go/auth/service/v1"
	"github.com/Servora-Kit/servora/app/servora/service/internal/biz"
	"github.com/Servora-Kit/servora/app/servora/service/internal/biz/entity"
)

// AuthService is a auth service.
type AuthService struct {
	authpb.UnimplementedAuthServiceServer

	uc *biz.AuthUsecase
}

// NewAuthService new a auth service.
func NewAuthService(uc *biz.AuthUsecase) *AuthService {
	return &AuthService{uc: uc}
}

func (s *AuthService) SignupByEmail(ctx context.Context, req *authpb.SignupByEmailRequest) (*authpb.SignupByEmailResponse, error) {
	// 参数校验
	if req.Password != req.PasswordConfirm {
		return nil, fmt.Errorf("password and confirm password do not match")
	}
	// 调用 biz 层
	user, err := s.uc.SignupByEmail(ctx, &entity.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}
	// 拼装返回结果
	return &authpb.SignupByEmailResponse{
		Id:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Role:  user.Role,
	}, nil
}

// LoginByEmailPassword user login by email and password.
func (s *AuthService) LoginByEmailPassword(ctx context.Context, req *authpb.LoginByEmailPasswordRequest) (*authpb.LoginByEmailPasswordResponse, error) {
	user := &entity.User{
		Email:    req.Email,
		Password: req.Password,
	}
	tokenPair, err := s.uc.LoginByEmailPassword(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("login by email password failed: %w", err)
	}
	return &authpb.LoginByEmailPasswordResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// RefreshToken refreshes the access token using a valid refresh token
func (s *AuthService) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
	tokenPair, err := s.uc.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token failed: %w", err)
	}
	return &authpb.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// Logout invalidates the refresh token
func (s *AuthService) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	err := s.uc.Logout(ctx, req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("logout failed: %w", err)
	}
	return &authpb.LogoutResponse{
		Success: true,
	}, nil
}
