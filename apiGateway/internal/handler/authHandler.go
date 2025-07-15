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

	if err := json.NewEncoder(w).Encode(tokens); err != nil {
		log.Println("[Login] error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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

	if err := json.NewEncoder(w).Encode(tokens); err != nil {
		log.Println("[Register] error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *authHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("refresh-token")
	if len(token) != 1 || token == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := a.gateway.Logout(r.Context(), token)
	if err != nil {
		log.Println("[Logout] error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
