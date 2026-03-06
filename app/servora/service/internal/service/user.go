package service

import (
	"context"
	"fmt"

	authpb "github.com/Servora-Kit/servora/api/gen/go/auth/service/v1"
	paginationpb "github.com/Servora-Kit/servora/api/gen/go/pagination/v1"
	userpb "github.com/Servora-Kit/servora/api/gen/go/user/service/v1"

	"github.com/Servora-Kit/servora/app/servora/service/internal/biz"
	"github.com/Servora-Kit/servora/app/servora/service/internal/biz/entity"
	"github.com/Servora-Kit/servora/app/servora/service/internal/consts"
)

type UserService struct {
	userpb.UnimplementedUserServiceServer

	uc *biz.UserUsecase
}

func NewUserService(uc *biz.UserUsecase) *UserService {
	return &UserService{uc: uc}
}

func (s *UserService) CurrentUserInfo(ctx context.Context, req *userpb.CurrentUserInfoRequest) (*userpb.CurrentUserInfoResponse, error) {
	user, err := s.uc.CurrentUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &userpb.CurrentUserInfoResponse{
		Id:   user.ID,
		Name: user.Name,
		Role: user.Role,
	}, nil
}

func (s *UserService) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	users, pagination, err := s.uc.ListUsers(ctx, req.GetPagination())
	if err != nil {
		return nil, err
	}

	respUsers := make([]*userpb.UserInfo, 0, len(users))
	for _, user := range users {
		respUsers = append(respUsers, &userpb.UserInfo{
			Id:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		})
	}

	if pagination == nil {
		pagination = &paginationpb.PaginationResponse{
			Mode: &paginationpb.PaginationResponse_Page{
				Page: &paginationpb.PagePaginationResponse{},
			},
		}
	}

	return &userpb.ListUsersResponse{
		Users:      respUsers,
		Pagination: pagination,
	}, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	currentUser, err := s.uc.CurrentUserInfo(ctx)
	if err != nil {
		return nil, err
	}

	switch currentUser.Role {
	case consts.User.String():
		if currentUser.ID != req.Id {
			return nil, authpb.ErrorUnauthorized("you only can update your own information")
		}
		if req.Role != "" && req.Role != consts.User.String() {
			return nil, authpb.ErrorUnauthorized("you do not have permission to change your role")
		}
	case consts.Admin.String():
		if req.Role != "" && req.Role >= consts.Admin.String() {
			return nil, authpb.ErrorUnauthorized("admin cannot assign role higher than admin")
		}
	case consts.Operator.String():
		if req.Role != "" && req.Role > consts.Operator.String() {
			return nil, authpb.ErrorUnauthorized("operator cannot assign role higher than operator")
		}
	}

	user := &entity.User{
		ID:       req.Id,
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	}
	_, err = s.uc.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return &userpb.UpdateUserResponse{
		Success: "true",
	}, nil
}

// SaveUser 保存用户
func (s *UserService) SaveUser(ctx context.Context, req *userpb.SaveUserRequest) (*userpb.SaveUserResponse, error) {
	user := &entity.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	}
	user, err := s.uc.SaveUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return &userpb.SaveUserResponse{Id: fmt.Sprintf("%d", user.ID)}, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	success, err := s.uc.DeleteUser(ctx, &entity.User{
		ID: req.Id,
	})
	if err != nil {
		return nil, err
	}
	return &userpb.DeleteUserResponse{Success: success}, err
}
