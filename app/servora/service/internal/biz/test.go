package biz

import (
	"context"

	"github.com/Servora-Kit/servora/pkg/logger"
)

type TestRepo interface {
	Hello(ctx context.Context, in string) (string, error)
}

type TestUsecase struct {
	repo TestRepo
	log  *logger.Helper
}

func NewTestUsecase(repo TestRepo, l logger.Logger) *TestUsecase {
	return &TestUsecase{
		repo: repo,
		log:  logger.NewHelper(l, logger.WithModule("test/biz/servora-service")),
	}
}

func (uc *TestUsecase) Hello(ctx context.Context, in string) (string, error) {
	uc.log.Debugf("Saying hello with greeting: %s", in)
	return uc.repo.Hello(ctx, in)
}
