package auth_transport_http

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	core_logger "github.com/NightFury2283/life_forge/internal/core/logger"
	core_http_response "github.com/NightFury2283/life_forge/internal/core/transport/http/response"
)

func (h *AuthHTTPHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		http.Error(w, "cannot initialize login", http.StatusInternalServerError)
		return
	}
	state := hex.EncodeToString(stateBytes)
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Value: state, Path: "/", MaxAge: 600,
		HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode})
	url := h.authService.GetGoogleAuthURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHTTPHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHanlder(log, w)
	w.Header().Set("Cache-Control", "no-store")
	stateCookie, stateErr := r.Cookie("oauth_state")
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode})
	state := r.URL.Query().Get("state")
	if stateErr != nil || state == "" || subtle.ConstantTimeCompare([]byte(state), []byte(stateCookie.Value)) != 1 {
		responseHandler.ErrorResponse(core_errors.ErrBadRequest, "invalid OAuth state; start login again")
		return
	}

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
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   24 * 60 * 60, //24 часа
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHTTPHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "jwt_token", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}
