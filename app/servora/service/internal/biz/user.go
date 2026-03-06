package biz

import (
	"context"

	authpb "github.com/Servora-Kit/servora/api/gen/go/auth/service/v1"
	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
	paginationpb "github.com/Servora-Kit/servora/api/gen/go/pagination/v1"
	userpb "github.com/Servora-Kit/servora/api/gen/go/user/service/v1"
	"github.com/Servora-Kit/servora/app/servora/service/internal/biz/entity"
	"github.com/Servora-Kit/servora/pkg/jwt"
	"github.com/Servora-Kit/servora/pkg/logger"
)

type UserRepo interface {
	SaveUser(context.Context, *entity.User) (*entity.User, error)
	GetUserById(context.Context, int64) (*entity.User, error)
	DeleteUser(context.Context, *entity.User) (*entity.User, error)
	UpdateUser(context.Context, *entity.User) (*entity.User, error)
	ListUsers(context.Context, int32, int32) ([]*entity.User, int64, error)
}

type UserUsecase struct {
	repo     UserRepo
	log      *logger.Helper
	cfg      *conf.App
	authRepo AuthRepo // 改为依赖 AuthRepo
}

func NewUserUsecase(repo UserRepo, l logger.Logger, cfg *conf.App, authRepo AuthRepo) *UserUsecase {
	uc := &UserUsecase{
		repo:     repo,
		log:      logger.NewHelper(l, logger.WithModule("user/biz/servora-service")),
		cfg:      cfg,
		authRepo: authRepo,
	}
	return uc
}

func (uc *UserUsecase) CurrentUserInfo(ctx context.Context) (*entity.User, error) {
	// 从context中获取当前登录用户信息
	claims, ok := jwt.FromContext[UserClaims](ctx)
	if !ok {
		return nil, authpb.ErrorUnauthorized("user not authenticated")
	}

	// 直接使用JWT中的信息构建用户模型，避免数据库查询
	user := &entity.User{
		ID:   claims.ID,
		Name: claims.Name,
		Role: claims.Role,
	}

	return user, nil
}

func (uc *UserUsecase) UpdateUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	// 获取原始用户信息
	origUser, err := uc.repo.GetUserById(ctx, user.ID)
	if err != nil {
		return nil, userpb.ErrorUserNotFound("user not found: %v", err)
	}

	// 只有当用户名发生变化时才检查重复
	if user.Name != origUser.Name {
		userWithSameName, err := uc.authRepo.GetUserByUserName(ctx, user.Name)
		if err != nil {
			return nil, authpb.ErrorUserNotFound("failed to check username: %v", err)
		}
		if userWithSameName != nil {
			return nil, authpb.ErrorUserAlreadyExists("username already exists")
		}
	}

	// 只有当邮箱发生变化时才检查重复
	if user.Email != origUser.Email {
		userWithSameEmail, err := uc.authRepo.GetUserByEmail(ctx, user.Email)
		if err != nil {
			return nil, authpb.ErrorUserNotFound("failed to check email: %v", err)
		}
		if userWithSameEmail != nil {
			return nil, authpb.ErrorUserAlreadyExists("email already exists")
		}
	}

	updatedUser, err := uc.repo.UpdateUser(ctx, user)
	if err != nil {
		return nil, userpb.ErrorUpdateUserFailed("failed to update user: %v", err)
	}
	return updatedUser, nil
}

func (uc *UserUsecase) SaveUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	if err := uc.checkUserExists(ctx, user); err != nil {
		return nil, err
	}

	savedUser, err := uc.repo.SaveUser(ctx, user)
	if err != nil {
		return nil, userpb.ErrorSaveUserFailed("failed to save user: %v", err)
	}
	return savedUser, nil
}

func (uc *UserUsecase) ListUsers(ctx context.Context, pagination *paginationpb.PaginationRequest) ([]*entity.User, *paginationpb.PaginationResponse, error) {
	page := int32(1)
	pageSize := int32(20)

	if pagination != nil {
		if pageMode := pagination.GetPage(); pageMode != nil {
			if pageMode.Page > 0 {
				page = pageMode.Page
			}
			if pageMode.PageSize > 0 {
				pageSize = pageMode.PageSize
			}
		}
	}

	users, total, err := uc.repo.ListUsers(ctx, page, pageSize)
	if err != nil {
		return nil, nil, userpb.ErrorUserNotFound("failed to list users: %v", err)
	}

	return users, &paginationpb.PaginationResponse{
		Mode: &paginationpb.PaginationResponse_Page{
			Page: &paginationpb.PagePaginationResponse{
				Total:    total,
				Page:     page,
				PageSize: pageSize,
			},
		},
	}, nil
}

func (uc *UserUsecase) DeleteUser(ctx context.Context, user *entity.User) (success bool, err error) {
	_, err = uc.repo.DeleteUser(ctx, user)
	if err != nil {
		return false, userpb.ErrorDeleteUserFailed("failed to delete user: %v", err)
	}
	return true, nil
}

func (uc *UserUsecase) checkUserExists(ctx context.Context, user *entity.User) error {
	if existingUser, err := uc.authRepo.GetUserByUserName(ctx, user.Name); err != nil {
		return authpb.ErrorUserNotFound("failed to check username: %v", err)
	} else if existingUser != nil {
		return authpb.ErrorUserAlreadyExists("username already exists")
	}

	if existingEmail, err := uc.authRepo.GetUserByEmail(ctx, user.Email); err != nil {
		return authpb.ErrorUserNotFound("failed to check email: %v", err)
	} else if existingEmail != nil {
		return authpb.ErrorUserAlreadyExists("email already exists")
	}
	return nil
}
