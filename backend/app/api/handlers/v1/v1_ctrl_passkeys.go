// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package v1

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/hay-kot/httpkit/server"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
)

const passkeyCookie = "zainaer.passkey"
const ceremonyTTL = 5 * time.Minute

type passkeyCeremony struct {
	session      webauthn.SessionData
	user         uuid.UUID
	binding      [32]byte
	name         string
	registration bool
	expires      time.Time
}
type passkeyRuntime struct {
	web     *webauthn.WebAuthn
	origin  string
	secure  bool
	mu      sync.Mutex
	pending map[string]passkeyCeremony
}

func newPasskeyRuntime(origin string) (*passkeyRuntime, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && !(u.Scheme == "http" && u.Hostname() == "localhost")) {
		return nil, errors.New("passkey origin must be an HTTPS origin (or http://localhost)")
	}
	origin = u.Scheme + "://" + u.Host
	web, err := webauthn.New(&webauthn.Config{RPID: u.Hostname(), RPDisplayName: "在哪儿 · Zainaer", RPOrigins: []string{origin},
		AttestationPreference:  protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{ResidentKey: protocol.ResidentKeyRequirementRequired, UserVerification: protocol.VerificationRequired},
	})
	if err != nil {
		return nil, err
	}
	return &passkeyRuntime{web: web, origin: origin, secure: u.Scheme == "https", pending: make(map[string]passkeyCeremony)}, nil
}
func (p *passkeyRuntime) save(w http.ResponseWriter, r *http.Request, c passkeyCeremony) error {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	id := base64.RawURLEncoding.EncodeToString(raw)
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.pending {
		if time.Now().After(v.expires) {
			delete(p.pending, k)
		}
	}
	if old, err := r.Cookie(passkeyCookie); err == nil {
		delete(p.pending, old.Value)
	}
	if len(p.pending) >= 4096 {
		return errors.New("too many pending ceremonies")
	}
	c.expires = time.Now().Add(ceremonyTTL)
	p.pending[id] = c
	http.SetCookie(w, &http.Cookie{Name: passkeyCookie, Value: id, Path: "/api/v1", MaxAge: 300, HttpOnly: true, Secure: p.secure, SameSite: http.SameSiteStrictMode})
	return nil
}
func (p *passkeyRuntime) take(w http.ResponseWriter, r *http.Request, registration bool, uid uuid.UUID) (passkeyCeremony, error) {
	http.SetCookie(w, &http.Cookie{Name: passkeyCookie, Path: "/api/v1", MaxAge: -1, HttpOnly: true, Secure: p.secure, SameSite: http.SameSiteStrictMode})
	cookie, err := r.Cookie(passkeyCookie)
	if err != nil {
		return passkeyCeremony{}, err
	}
	p.mu.Lock()
	c, ok := p.pending[cookie.Value]
	delete(p.pending, cookie.Value)
	p.mu.Unlock()
	if !ok || time.Now().After(c.expires) || c.registration != registration || c.user != uid {
		return c, errors.New("invalid or expired ceremony")
	}
	if registration && c.binding != passkeySessionBinding(r) {
		return c, errors.New("session changed")
	}
	return c, nil
}
func passkeySessionBinding(r *http.Request) [32]byte {
	c, err := r.Cookie(cookieNameToken)
	if err != nil {
		return [32]byte{}
	}
	return sha256.Sum256([]byte(c.Value))
}

type passkeyUser struct {
	repo.UserOut
	credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte                         { return u.ID[:] }
func (u *passkeyUser) WebAuthnName() string                       { return u.Email }
func (u *passkeyUser) WebAuthnDisplayName() string                { return u.Name }
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }
func (ctrl *V1Controller) loadPasskeyUser(r *http.Request, uid uuid.UUID) (*passkeyUser, error) {
	u, err := ctrl.repo.Users.GetOneID(r.Context(), uid)
	if err != nil {
		return nil, err
	}
	rows, err := ctrl.repo.Users.Passkeys(r.Context(), uid)
	if err != nil {
		return nil, err
	}
	result := &passkeyUser{UserOut: u}
	for _, row := range rows {
		var c webauthn.Credential
		if err = json.Unmarshal(row.Credential, &c); err != nil {
			return nil, err
		}
		result.credentials = append(result.credentials, c)
	}
	return result, nil
}
func (ctrl *V1Controller) passkeyGuard(w http.ResponseWriter, r *http.Request, manage bool) error {
	w.Header().Set("Cache-Control", "no-store")
	if ctrl.passkeys == nil || ctrl.isDemo || !ctrl.config.Options.AllowLocalLogin {
		return validate.NewRequestError(errors.New("passkeys unavailable"), http.StatusServiceUnavailable)
	}
	if r.Method != http.MethodGet && r.Header.Get("Origin") != ctrl.passkeys.origin {
		return validate.NewRequestError(errors.New("invalid origin"), http.StatusForbidden)
	}
	if manage && (services.IsAPIKeyAuth(r.Context()) || services.UseUserCtx(r.Context()) == nil || passkeySessionBinding(r) == [32]byte{}) {
		return validate.NewUnauthorizedError()
	}
	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	return nil
}
func (ctrl *V1Controller) HandlePasskeyBegin(register bool) errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := ctrl.passkeyGuard(w, r, register); err != nil {
			return err
		}
		var options interface{}
		var session *webauthn.SessionData
		var err error
		c := passkeyCeremony{registration: register}
		if register {
			c.user = services.UseUserCtx(r.Context()).ID
			c.binding = passkeySessionBinding(r)
			var in struct {
				Name     string `json:"name"`
				Password string `json:"password"`
			}
			if server.Decode(r, &in) != nil || len(strings.TrimSpace(in.Name)) == 0 || len(in.Name) > 80 || len(in.Password) > 1024 {
				return validate.NewRequestError(errors.New("invalid passkey details"), 400)
			}
			if ctrl.svc.User.VerifyPasskeyPassword(r.Context(), c.user, in.Password) != nil {
				return validate.NewUnauthorizedError()
			}
			user, e := ctrl.loadPasskeyUser(r, c.user)
			if e != nil {
				return validate.NewUnauthorizedError()
			}
			if len(user.credentials) >= 20 {
				return validate.NewRequestError(errors.New("passkey limit reached"), 409)
			}
			c.name = strings.TrimSpace(in.Name)
			exclude := make([]protocol.CredentialDescriptor, 0, len(user.credentials))
			for _, key := range user.credentials {
				exclude = append(exclude, key.Descriptor())
			}
			options, session, err = ctrl.passkeys.web.BeginRegistration(user, webauthn.WithExclusions(exclude))
		} else {
			options, session, err = ctrl.passkeys.web.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
		}
		if err != nil {
			return validate.NewRequestError(errors.New("cannot start passkey request"), 500)
		}
		c.session = *session
		if err = ctrl.passkeys.save(w, r, c); err != nil {
			return validate.NewRequestError(errors.New("try again later"), 503)
		}
		return server.JSON(w, 200, options)
	}
}
func (ctrl *V1Controller) HandlePasskeyFinish(register bool) errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := ctrl.passkeyGuard(w, r, register); err != nil {
			return err
		}
		uid := uuid.Nil
		if register {
			uid = services.UseUserCtx(r.Context()).ID
		}
		c, err := ctrl.passkeys.take(w, r, register, uid)
		if err != nil {
			return validate.NewUnauthorizedError()
		}
		if register {
			u, err := ctrl.loadPasskeyUser(r, uid)
			if err != nil {
				return validate.NewUnauthorizedError()
			}
			key, err := ctrl.passkeys.web.FinishRegistration(u, c.session, r)
			if err != nil {
				return validate.NewUnauthorizedError()
			}
			if err = ctrl.repo.Users.AddPasskey(r.Context(), uid, c.name, key); err != nil {
				return validate.NewRequestError(errors.New("cannot save passkey"), 409)
			}
			return server.JSON(w, 201, map[string]bool{"ok": true})
		}
		var stored *ent.Passkey
		user, key, err := ctrl.passkeys.web.FinishPasskeyLogin(func(rawID, handle []byte) (webauthn.User, error) {
			uid, e := uuid.FromBytes(handle)
			if e != nil {
				return nil, e
			}
			stored, e = ctrl.repo.Users.FindPasskey(r.Context(), rawID, uid)
			if e != nil {
				return nil, e
			}
			return ctrl.loadPasskeyUser(r, uid)
		}, c.session, r)
		if err != nil || stored == nil || key.Authenticator.CloneWarning {
			return validate.NewUnauthorizedError()
		}
		if err = ctrl.repo.Users.UsePasskey(r.Context(), stored, key); err != nil {
			return validate.NewUnauthorizedError()
		}
		token, err := ctrl.svc.User.LoginPasskey(r.Context(), user.(*passkeyUser).ID)
		if err != nil {
			return validate.NewUnauthorizedError()
		}
		ctrl.setCookies(w, noPort(r.Host), token.Raw, token.ExpiresAt, true, token.AttachmentToken)
		return server.JSON(w, 200, TokenResponse{Token: "Bearer " + token.Raw, ExpiresAt: token.ExpiresAt, AttachmentToken: token.AttachmentToken})
	}
}
func (ctrl *V1Controller) HandlePasskeyList() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := ctrl.passkeyGuard(w, r, true); err != nil {
			return err
		}
		rows, err := ctrl.repo.Users.Passkeys(r.Context(), services.UseUserCtx(r.Context()).ID)
		if err != nil {
			return err
		}
		type summary struct {
			ID         uuid.UUID  `json:"id"`
			Name       string     `json:"name"`
			CreatedAt  time.Time  `json:"createdAt"`
			LastUsedAt *time.Time `json:"lastUsedAt"`
		}
		out := make([]summary, 0, len(rows))
		for _, v := range rows {
			out = append(out, summary{v.ID, v.Name, v.CreatedAt, v.LastUsedAt})
		}
		return server.JSON(w, 200, out)
	}
}
func (ctrl *V1Controller) HandlePasskeyDelete() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := ctrl.passkeyGuard(w, r, true); err != nil {
			return err
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			return validate.NewRequestError(errors.New("invalid id"), 400)
		}
		if err = ctrl.repo.Users.DeletePasskey(r.Context(), services.UseUserCtx(r.Context()).ID, id); err != nil {
			return validate.NewRequestError(errors.New("passkey not found"), 404)
		}
		return server.JSON(w, 204, nil)
	}
}
