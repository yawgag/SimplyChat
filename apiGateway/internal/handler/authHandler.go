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

	setTokensCookie(w, tokens)
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
	setTokensCookie(w, tokens)
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

func setTokensCookie(w http.ResponseWriter, tokens *models.AuthTokens) {
	if tokens.AccessToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "access_token",
			Value:    tokens.AccessToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   15 * 60,
		})
	}
	if tokens.RefreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    tokens.RefreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
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
