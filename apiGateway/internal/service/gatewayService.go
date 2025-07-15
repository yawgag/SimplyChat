package service

import (
	"apiGateway/internal/models"
	"context"
)

type GatewayService interface {
	Login(ctx context.Context, user *models.User) (*models.AuthTokens, error)
	Register(ctx context.Context, user *models.User) (*models.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	UpdateTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error)
}

type gatewayService struct {
	auth AuthService
}

func NewGatewayService(auth AuthService) GatewayService {
	out := &gatewayService{
		auth: auth,
	}
	return out
}

func (g *gatewayService) Login(ctx context.Context, user *models.User) (*models.AuthTokens, error) {
	tokens, err := g.auth.Login(ctx, user)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func (g *gatewayService) Register(ctx context.Context, user *models.User) (*models.AuthTokens, error) {
	tokens, err := g.auth.Register(ctx, user)
	if err != nil {
		return nil, err
	}

	return tokens, err
}

func (g *gatewayService) Logout(ctx context.Context, refreshToken string) error {
	err := g.auth.Logout(ctx, refreshToken)
	return err
}

func (g *gatewayService) UpdateTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error) {
	tokens, err := g.auth.UpdateTokens(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}
