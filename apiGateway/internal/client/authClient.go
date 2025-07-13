package client

import (
	pb "apiGateway/internal/gen/auth"
)

type grpcClient struct {
	client pb.AuthServiceClient
}
