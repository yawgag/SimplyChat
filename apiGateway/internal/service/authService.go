package service

import (
	"apiGateway/internal/client"
	"apiGateway/internal/models"
	"context"
)

type AuthService interface {
	Login(ctx context.Context, user *models.User) (*models.AuthTokens, error)
	Register(ctx context.Context, user *models.User) (*models.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	UpdateTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error)
}

type authService struct {
	client client.AuthClient
}

func NewAuthService(client client.AuthClient) AuthService {
	out := &authService{
		client: client,
	}
	return out
}

func (a *authService) Login(ctx context.Context, user *models.User) (*models.AuthTokens, error) {
	tokens, err := a.client.Login(ctx, user)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (a *authService) Register(ctx context.Context, user *models.User) (*models.AuthTokens, error) {
	tokens, err := a.client.Register(ctx, user)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (a *authService) Logout(ctx context.Context, refreshToken string) error {
	err := a.client.Logout(ctx, refreshToken)
	return err
}

func (a *authService) UpdateTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error) {
	tokens, err := a.client.UpdateTokens(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	return tokens, err
}
