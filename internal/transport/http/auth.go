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

const (
	userKey   ctxKey = 0
	orgObjKey ctxKey = 2 // domain.Org (orgKey=1 holds the OrgID, set by both auth + api-key)
)

type authHandler struct {
	log    *slog.Logger
	svc    *auth.Service
	orgs   domain.OrgStore
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
	h.setSessionCookie(w, token, su.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{"user_id": su.UserID, "phone": su.Phone})
}

// setSessionCookie writes the httpOnly session cookie (secure in prod).
func (h *authHandler) setSessionCookie(w http.ResponseWriter, token string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  exp,
	})
}

type credsBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// register creates an email/password account and logs it in.
func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var body credsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	token, su, err := h.svc.Register(r.Context(), body.Email, body.Password, body.Name)
	if errors.Is(err, domain.ErrEmailTaken) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
		return
	}
	if errors.Is(err, domain.ErrInvalidCredentials) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and an 8+ character password are required"})
		return
	}
	if err != nil {
		h.log.Error("register", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	h.setSessionCookie(w, token, su.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{"user_id": su.UserID, "email": su.Email})
}

// login verifies email/password and issues a session.
func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var body credsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	token, su, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if errors.Is(err, domain.ErrInvalidCredentials) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}
	if err != nil {
		h.log.Error("login", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	h.setSessionCookie(w, token, su.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{"user_id": su.UserID, "email": su.Email})
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
	org, _ := r.Context().Value(orgObjKey).(domain.Org)
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": su.UserID,
		"phone":   su.Phone,
		"email":   su.Email,
		"name":    su.Name,
		"org":     map[string]any{"id": string(org.ID), "name": org.Name},
	})
}

// renameOrg updates the caller's org display name.
func (h *authHandler) renameOrg(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	if err := h.orgs.Rename(r.Context(), orgFromCtx(r), body.Name); err != nil {
		h.log.Error("rename org", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": body.Name})
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
		org, err := h.orgs.EnsureForUser(r.Context(), su.UserID, "My Store")
		if err != nil {
			h.log.Error("ensure org", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		ctx := context.WithValue(r.Context(), userKey, su)
		ctx = context.WithValue(ctx, orgKey, org.ID)
		ctx = context.WithValue(ctx, orgObjKey, org)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
