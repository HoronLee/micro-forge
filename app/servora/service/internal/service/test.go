package service

import (
	"context"

	testpb "github.com/Servora-Kit/servora/api/gen/go/test/service/v1"
	"github.com/Servora-Kit/servora/app/servora/service/internal/biz"
)

// TestService is a test service.
type TestService struct {
	testpb.UnimplementedTestServiceServer

	uc *biz.TestUsecase
}

// NewTestService new a test service.
func NewTestService(uc *biz.TestUsecase) *TestService {
	return &TestService{uc: uc}
}

// Hello calls the hello service
func (s *TestService) Hello(ctx context.Context, req *testpb.HelloRequest) (*testpb.HelloResponse, error) {
	// 调用 biz 层
	res, err := s.uc.Hello(ctx, req.Req)
	if err != nil {
		return nil, err
	}
	// 拼装返回响应
	return &testpb.HelloResponse{
		Rep: res,
	}, nil
}

// Test is a test method.
func (s *TestService) Test(ctx context.Context, req *testpb.TestRequest) (*testpb.TestResponse, error) {
	return &testpb.TestResponse{Message: "公开的测试路由"}, nil
}

// PrivateTest is a private test method.
func (s *TestService) PrivateTest(ctx context.Context, req *testpb.PrivateTestRequest) (*testpb.PrivateTestResponse, error) {
	return &testpb.PrivateTestResponse{Message: "私有的测试路由"}, nil
}
