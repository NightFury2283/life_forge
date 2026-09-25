package auth_transport_http

import (
	"fmt"
	"net/http"

	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	core_logger "github.com/NightFury2283/life_forge/internal/core/logger"
	core_http_response "github.com/NightFury2283/life_forge/internal/core/transport/http/response"
)

func (h *AuthHTTPHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := h.authService.GetGoogleAuthURL()
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHTTPHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHanlder(log, w)

	code := r.URL.Query().Get("code")
	if code == "" {
		responseHandler.ErrorResponse(core_errors.ErrBadRequest, "get code from query")
		return
	}

	jwtToken, err := h.authService.LoginViaGoogle(ctx, code)
	if err != nil {
		responseHandler.ErrorResponse(err, "get jwt token from service")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    jwtToken,
		HttpOnly: true,
		Secure:   false, //TODO поменять на true в проде (с HTTPS)
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   24 * 60 * 60, //24 часа
	})

	//сборка адреса редиректа
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	redirectURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	http.Redirect(w, r, redirectURL, http.StatusPermanentRedirect)
}
