package data

import (
	"context"

	sayhellopb "github.com/Servora-Kit/servora/api/gen/go/sayhello/service/v1"
	"github.com/Servora-Kit/servora/app/servora/service/internal/biz"
	"github.com/Servora-Kit/servora/pkg/logger"
	"github.com/Servora-Kit/servora/pkg/transport/client"
)

type testRepo struct {
	data *Data
	log  *logger.Helper
}

func NewTestRepo(data *Data, l logger.Logger) biz.TestRepo {
	return &testRepo{
		data: data,
		log:  logger.NewHelper(l, logger.WithModule("test/data/servora-service")),
	}
}

func (r *testRepo) Hello(ctx context.Context, in string) (string, error) {
	r.log.Debugf("Saying hello with greeting: %s", in)

	conn, err := client.GetGRPCConn(ctx, r.data.client, "sayhello.service")
	if err != nil {
		r.log.Errorf("Failed to create grpc connection: %v", err)
		return "", err
	}

	helloClient := sayhellopb.NewSayHelloServiceClient(conn)
	ret, err := helloClient.Hello(ctx, &sayhellopb.HelloRequest{Greeting: in})
	if err != nil {
		r.log.Errorf("Failed to say hello: %v", err)
		return "", err
	}
	return ret.GetReply(), nil
}
