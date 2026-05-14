package controller

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/anupam-chopra/prism/internal/api/request"
	"github.com/anupam-chopra/prism/internal/api/response"
	"github.com/anupam-chopra/prism/internal/logger"
	"github.com/anupam-chopra/prism/internal/service"
)

type TokenController struct {
	service service.TokenServiceI
	logger  *slog.Logger
}

func NewTokenController(service service.TokenServiceI, appLogger *slog.Logger) *TokenController {
	if service == nil {
		panic("service cannot be nil")
	}
	if appLogger == nil {
		panic("logger cannot be nil")
	}

	return &TokenController{
		service: service,
		logger:  appLogger,
	}
}

func (c *TokenController) Issue(w http.ResponseWriter, r *http.Request) {
	var req request.TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := req.Validate(); err != nil {
		response.ValidationError(w, err)
		return
	}

	token, err := c.service.Issue(r.Context(), req.ToDomain())
	if err != nil {
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	response.TokenCreated(w, token)
}

func (c *TokenController) Revoke(w http.ResponseWriter, r *http.Request) {
	var req request.RevokeTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := req.Validate(); err != nil {
		response.ValidationError(w, err)
		return
	}

	if err := c.service.Revoke(r.Context(), req.JTI); err != nil {
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	response.Success(w, http.StatusOK, map[string]string{
		"status": "revoked",
		"jti":    req.JTI,
	})
}
