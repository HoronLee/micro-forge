package service

import (
	"context"
	"fmt"

	sayhellov1 "github.com/Servora-Kit/servora/api/gen/go/sayhello/service/v1"
)

type SayHelloService struct {
	sayhellov1.UnimplementedSayHelloServiceServer
}

func NewSayHelloService() *SayHelloService {
	return &SayHelloService{}
}

func (s *SayHelloService) Hello(ctx context.Context, req *sayhellov1.HelloRequest) (*sayhellov1.HelloResponse, error) {
	greeting := req.Greeting
	return &sayhellov1.HelloResponse{
		Reply: fmt.Sprintf("Hello, %s!", greeting),
	}, nil
}
