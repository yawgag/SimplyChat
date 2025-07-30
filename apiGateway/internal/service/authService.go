package service

import (
	"apiGateway/internal/client"
	"apiGateway/internal/models"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthService interface {
	Login(ctx context.Context, user *models.User) (*models.AuthTokens, error)
	Register(ctx context.Context, user *models.User) (*models.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
}

func NewAuthService(client client.AuthClient) AuthService {
	out := &authService{
		client: client,
	}
	return out
}

type TokensHandler interface {
	UpdateTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error)
	ValidateAccessToken(tokenString string) (*models.AccessToken, error)
}

func NewTokensHadnler(client client.AuthClient) TokensHandler {
	publicKeyPem, err := client.GetPublicRSAKey(context.Background())
	if err != nil {
		log.Fatal("[NewTokensHadnler] internal error: ", err)
	}

	block, _ := pem.Decode([]byte(publicKeyPem))
	if block == nil {
		log.Fatal("[NewTokensHadnler] internal error")
	}

	pubKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		log.Fatal("[NewTokensHadnler] internal error: ", err)
	}

	out := &authService{
		client:       client,
		publicRsaKey: pubKey,
	}
	return out
}

type authService struct {
	client       client.AuthClient
	publicRsaKey *rsa.PublicKey
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

func (a *authService) ValidateAccessToken(tokenString string) (*models.AccessToken, error) {
	token := strings.TrimPrefix(tokenString, "Bearer ")
	jwtToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodRSA)
		if !ok {
			return nil, fmt.Errorf("[ValidateAccessToken] error: wrong token format")
		}
		return a.publicRsaKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if ok && jwtToken.Valid {

		uidStr, ok := (claims)["uid"].(string)
		if !ok {
			return nil, fmt.Errorf("[ValidateAccessToken] error: bad access token")
		}
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			return nil, fmt.Errorf("[ValidateAccessToken] error: bad access token")
		}

		userRole, ok := (claims)["userRole"].(string)
		if !ok {
			return nil, fmt.Errorf("[ValidateAccessToken] error: bad access token")
		}

		expFloat, ok := (claims)["exp"].(float64)
		if !ok {
			return nil, fmt.Errorf("[ValidateAccessToken] error: bad access token")
		}
		exp := int64(expFloat)

		if exp < time.Now().Unix() {
			return nil, fmt.Errorf("[ValidateAccessToken] error: token is expired")
		}

		outToken := &models.AccessToken{
			Uid:      uid,
			UserRole: userRole,
			Exp:      exp,
		}

		return outToken, nil
	}

	return nil, errors.New("wrong token")

}
