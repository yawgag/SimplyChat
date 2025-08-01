package models

import (
	"github.com/google/uuid"
)

type AuthTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type AccessToken struct {
	Uid      uuid.UUID
	UserRole string
	Exp      int64
}

type User struct {
	Uid      *uuid.UUID `json:"uid"`
	Login    *string    `json:"login"`
	Email    *string    `json:"email"`
	Password *string    `json:"password"`
	Role     *string    `json:"role"`
}
