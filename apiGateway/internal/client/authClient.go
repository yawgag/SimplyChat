package client

import (
	pb "apiGateway/internal/gen/auth"
	"apiGateway/internal/models"
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

type grpcClient struct {
	client pb.AuthServiceClient
}

type AuthClient interface {
	Login(ctx context.Context, user *models.User) (*models.AuthTokens, error)
	Register(ctx context.Context, user *models.User) (*models.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error

	GetPublicRSAKey(ctx context.Context) (string, error)                               // internal method
	UpdateTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error) // internal method
}

func NewAuthClient(conn *grpc.ClientConn) AuthClient {
	out := &grpcClient{
		client: pb.NewAuthServiceClient(conn),
	}
	return out
}

func (g *grpcClient) Login(ctx context.Context, user *models.User) (*models.AuthTokens, error) {
	req := &pb.LoginRequest{
		Login:    *user.Login,
		Password: *user.Password,
	}
	pbTokens, err := g.client.Login(ctx, req)
	if err != nil {
		return nil, err
	}

	out := &models.AuthTokens{
		AccessToken:  pbTokens.AccessToken,
		RefreshToken: pbTokens.RefreshToken,
	}

	return out, nil
}
func (g *grpcClient) Register(ctx context.Context, user *models.User) (*models.AuthTokens, error) {
	req := &pb.RegisterRequest{
		Login:    *user.Login,
		Password: *user.Password,
		Email:    *user.Email,
	}

	pbTokens, err := g.client.Register(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &models.AuthTokens{
		AccessToken:  pbTokens.AccessToken,
		RefreshToken: pbTokens.RefreshToken,
	}

	return out, nil
}

func (g *grpcClient) Logout(ctx context.Context, refreshToken string) error {
	md := metadata.New(map[string]string{
		"refresh-token": refreshToken,
	})

	ctxWithMetadata := metadata.NewOutgoingContext(ctx, md)

	_, err := g.client.Logout(ctxWithMetadata, &emptypb.Empty{})

	return err
}

// internal method only. NOT FOR USER
func (g *grpcClient) GetPublicRSAKey(ctx context.Context) (string, error) {
	publicKey, err := g.client.GetPublicRSAKey(ctx, &emptypb.Empty{})
	if err != nil {
		return "", err
	}

	return publicKey.PublicKey, nil
}

func (g *grpcClient) UpdateTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error) {
	md := metadata.New(map[string]string{
		"refresh-token": refreshToken,
	})
	ctxWithMetadata := metadata.NewOutgoingContext(ctx, md)

	pbTokens, err := g.client.UpdateTokens(ctxWithMetadata, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	out := &models.AuthTokens{
		AccessToken:  pbTokens.AccessToken,
		RefreshToken: pbTokens.RefreshToken,
	}
	return out, nil
}
