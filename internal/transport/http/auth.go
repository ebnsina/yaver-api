package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/service/auth"
	"github.com/ebnsina/yaver-api/pkg/phone"
)

const cookieName = "yaver_session"

type ctxKey int

const userKey ctxKey = 0

type authHandler struct {
	log    *slog.Logger
	svc    *auth.Service
	orgs   domain.OrgProvisioner
	secure bool // set the Secure cookie flag outside dev
}

type otpRequestBody struct {
	Phone string `json:"phone"`
}

func (h *authHandler) requestOTP(w http.ResponseWriter, r *http.Request) {
	var body otpRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	e164, err := phone.NormalizeBD(body.Phone)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid phone"})
		return
	}
	devCode, err := h.svc.RequestOTP(r.Context(), e164)
	if err != nil {
		h.log.Error("request otp", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	resp := map[string]string{"status": "sent"}
	if devCode != "" { // dev only: no SMS provider yet
		resp["dev_code"] = devCode
	}
	writeJSON(w, http.StatusOK, resp)
}

type otpVerifyBody struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

func (h *authHandler) verifyOTP(w http.ResponseWriter, r *http.Request) {
	var body otpVerifyBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	e164, err := phone.NormalizeBD(body.Phone)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid phone"})
		return
	}
	token, su, err := h.svc.VerifyOTP(r.Context(), e164, body.Code)
	if errors.Is(err, domain.ErrInvalidOTP) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired code"})
		return
	}
	if err != nil {
		h.log.Error("verify otp", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  su.ExpiresAt,
	})
	writeJSON(w, http.StatusOK, map[string]any{"user_id": su.UserID, "phone": su.Phone})
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		_ = h.svc.Logout(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(0, 0)})
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	su, _ := r.Context().Value(userKey).(domain.SessionUser)
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": su.UserID,
		"phone":   su.Phone,
		"email":   su.Email,
		"name":    su.Name,
	})
}

// requireAuth resolves the session cookie and injects the user, or 401s.
func (h *authHandler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		su, found, err := h.svc.Resolve(r.Context(), c.Value)
		if err != nil {
			h.log.Error("resolve session", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		if !found {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		// Resolve (and lazily provision) the user's org — auto-provision on first
		// authenticated request. Every authed handler reads the org from context.
		orgID, err := h.orgs.EnsureForUser(r.Context(), su.UserID, "My Store")
		if err != nil {
			h.log.Error("ensure org", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		ctx := context.WithValue(r.Context(), userKey, su)
		ctx = context.WithValue(ctx, orgKey, orgID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
