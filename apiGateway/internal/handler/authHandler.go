package handler

import (
	"apiGateway/internal/models"
	"apiGateway/internal/service"
	"encoding/json"
	"log"
	"net/http"
)

type authHandler struct {
	gateway service.GatewayService
}

func NewAuthHandler(gateway service.GatewayService) *authHandler {
	out := &authHandler{
		gateway: gateway,
	}
	return out
}

func (a *authHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		log.Println("[Login] error: wrong content-type")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var user *models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Println("[Login] error: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// validation of data
	if user.Login == nil || user.Password == nil {
		log.Println("[Login] error: not enough data")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tokens, err := a.gateway.Login(r.Context(), user)
	if err != nil {
		log.Println("[Login] error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	setAccessTokenCookie(w, tokens.AccessToken)
	setRefreshTokenCookie(w, tokens.RefreshToken)
}

func (a *authHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		log.Println("[Register] error: wrong content-type")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var user *models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Println("[Register] error: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// validation of data
	if user.Login == nil || user.Password == nil || user.Email == nil {
		log.Println("[Register] error: not enough data")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tokens, err := a.gateway.Register(r.Context(), user)
	if err != nil {
		log.Println("[Register] error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	setAccessTokenCookie(w, tokens.AccessToken)
	setRefreshTokenCookie(w, tokens.RefreshToken)
}

func (a *authHandler) Logout(w http.ResponseWriter, r *http.Request) {

	accessTokenCookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if accessTokenCookie.Value == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = a.gateway.Logout(r.Context(), accessTokenCookie.Value)
	if err != nil {
		log.Println("[Logout] error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	deleteCookie(w, "access_token")
	deleteCookie(w, "refresh_token")
}

func (a *authHandler) HandleCheckAuth(w http.ResponseWriter, r *http.Request) {
	accessTokenCookie, err := r.Cookie("access_token")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "No access token", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Error reading cookie", http.StatusInternalServerError)
		return
	}

	if accessTokenCookie.Value == "" {
		http.Error(w, "Empty access token", http.StatusUnauthorized)
		return
	}

	_, err = a.gateway.ValidateAccessToken(accessTokenCookie.Value)
	if err != nil {
		http.Error(w, "Invalid access token", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *authHandler) HandleUpdateTokens(w http.ResponseWriter, r *http.Request) {
	refershTokenCookie, err := r.Cookie("refresh_token")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tokens, err := a.gateway.UpdateTokens(r.Context(), refershTokenCookie.Value)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	setAccessTokenCookie(w, tokens.AccessToken)
	w.WriteHeader(http.StatusOK)
}

func setAccessTokenCookie(w http.ResponseWriter, accessToken string) {
	if accessToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "access_token",
			Value:    accessToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   15 * 60,
		})
	}
}

func setRefreshTokenCookie(w http.ResponseWriter, refreshToken string) {
	if refreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   30 * 24 * 60 * 60,
		})
	}
}

func deleteCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
