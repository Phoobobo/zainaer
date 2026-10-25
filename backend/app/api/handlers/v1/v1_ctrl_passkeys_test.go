// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package v1

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"github.com/sysadminsmedia/homebox/backend/pkgs/hasher"
)

func TestPasskeySignedLifecycle(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := ent.Open("sqlite3", "file:"+filepath.Join(dir, "passkeys.db")+"?_fk=1&_time_format=sqlite")
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.Schema.Create(ctx))
	repos := repo.New(db, nil, config.Storage{PrefixPath: dir, ConnString: "file://" + dir}, "mem://{{ .Topic }}", config.Thumbnail{})
	svc := services.New(repos, services.WithAutoIncrementAssetID(false))
	u, err := svc.User.RegisterUser(ctx, services.UserRegistration{Name: "Passkey user", Email: "keys@example.test", Password: "correct-password"})
	require.NoError(t, err)
	ctrl := &V1Controller{repo: repos, svc: svc, config: &config.Config{Options: config.Options{AllowLocalLogin: true}}}
	ctrl.passkeys, err = newPasskeyRuntime("https://app.example.test")
	require.NoError(t, err)
	authed := services.SetUserCtx(ctx, &u, "session-one")
	request := func(body any, manage bool, cookie *http.Cookie) *http.Request {
		b, e := json.Marshal(body)
		require.NoError(t, e)
		r := httptest.NewRequest("POST", "https://app.example.test/api/v1/passkey", bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", ctrl.passkeys.origin)
		if manage {
			r = r.WithContext(authed)
			r.AddCookie(&http.Cookie{Name: cookieNameToken, Value: "session-one"})
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		return r
	}
	begin := func(register bool) (string, *http.Cookie) {
		w := httptest.NewRecorder()
		r := request(map[string]string{"name": "My device", "password": "correct-password"}, register, nil)
		require.NoError(t, ctrl.HandlePasskeyBegin(register)(w, r))
		var out struct {
			PublicKey struct {
				Challenge string `json:"challenge"`
			} `json:"publicKey"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
		require.NotEmpty(t, out.PublicKey.Challenge)
		cookie := w.Result().Cookies()[0]
		require.True(t, cookie.HttpOnly)
		require.True(t, cookie.Secure)
		require.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
		return out.PublicKey.Challenge, cookie
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	id := make([]byte, 32)
	_, err = rand.Read(id)
	require.NoError(t, err)
	enc := base64.RawURLEncoding.EncodeToString
	client := func(challenge, kind, origin string) []byte {
		b, e := json.Marshal(map[string]any{"type": kind, "challenge": challenge, "origin": origin, "crossOrigin": false})
		require.NoError(t, e)
		return b
	}
	authData := func(flags byte, counter uint32) []byte {
		rp := sha256.Sum256([]byte("app.example.test"))
		b := append([]byte{}, rp[:]...)
		b = append(b, flags)
		return binary.BigEndian.AppendUint32(b, counter)
	}
	registration := func(challenge string) map[string]any {
		pub, e := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: key.X.FillBytes(make([]byte, 32)), -3: key.Y.FillBytes(make([]byte, 32))})
		require.NoError(t, e)
		ad := append(authData(0x45, 0), make([]byte, 16)...)
		ad = binary.BigEndian.AppendUint16(ad, uint16(len(id)))
		ad = append(ad, id...)
		ad = append(ad, pub...)
		att, e := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": ad})
		require.NoError(t, e)
		return map[string]any{"id": enc(id), "rawId": enc(id), "type": "public-key", "clientExtensionResults": map[string]any{}, "response": map[string]any{"clientDataJSON": enc(client(challenge, "webauthn.create", ctrl.passkeys.origin)), "attestationObject": enc(att), "transports": []string{"internal"}}}
	}
	assertion := func(challenge, origin string, flags byte, counter uint32, handle uuid.UUID, tamper bool) map[string]any {
		cd := client(challenge, "webauthn.get", origin)
		ad := authData(flags, counter)
		h := sha256.Sum256(cd)
		payload := append(append([]byte{}, ad...), h[:]...)
		digest := sha256.Sum256(payload)
		sig, e := ecdsa.SignASN1(rand.Reader, key, digest[:])
		require.NoError(t, e)
		if tamper {
			sig[len(sig)-1] ^= 1
		}
		return map[string]any{"id": enc(id), "rawId": enc(id), "type": "public-key", "clientExtensionResults": map[string]any{}, "response": map[string]any{"clientDataJSON": enc(cd), "authenticatorData": enc(ad), "signature": enc(sig), "userHandle": enc(handle[:])}}
	}
	t.Run("step-up and origin required", func(t *testing.T) {
		apiRequest := request(map[string]string{"name": "key", "password": "correct-password"}, true, nil)
		apiRequest = apiRequest.WithContext(services.SetAPIKeyAuth(authed))
		require.Error(t, ctrl.HandlePasskeyBegin(true)(httptest.NewRecorder(), apiRequest), "API keys cannot register credentials")

		require.Error(t, ctrl.HandlePasskeyBegin(true)(httptest.NewRecorder(), request(map[string]string{"name": "bad", "password": "wrong"}, true, nil)))
		r := request(nil, false, nil)
		r.Header.Set("Origin", "https://evil.example")
		require.Error(t, ctrl.HandlePasskeyBegin(false)(httptest.NewRecorder(), r))
		require.Error(t, ctrl.HandlePasskeyBegin(true)(httptest.NewRecorder(), request(nil, false, nil)))
	})
	challenge, cookie := begin(true)
	body := registration(challenge)
	require.NoError(t, ctrl.HandlePasskeyFinish(true)(httptest.NewRecorder(), request(body, true, cookie)))
	require.Error(t, ctrl.HandlePasskeyFinish(true)(httptest.NewRecorder(), request(body, true, cookie)), "registration replay")
	stored, err := repos.Users.FindPasskey(ctx, id, u.ID)
	require.NoError(t, err)
	challenge, cookie = begin(false)
	body = assertion(challenge, ctrl.passkeys.origin, 0x05, 1, u.ID, false)
	w := httptest.NewRecorder()
	require.NoError(t, ctrl.HandlePasskeyFinish(false)(w, request(body, false, cookie)))
	var token TokenResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &token))
	require.NotEmpty(t, token.Token)
	authenticated, err := repos.AuthTokens.GetUserFromToken(ctx, hasher.HashToken(token.Token[7:]))
	require.NoError(t, err)
	require.Equal(t, u.ID, authenticated.ID)
	require.Error(t, ctrl.HandlePasskeyFinish(false)(httptest.NewRecorder(), request(body, false, cookie)), "assertion replay")
	for _, tc := range []struct {
		name, origin string
		flags        byte
		counter      uint32
		handle       uuid.UUID
		tamper       bool
	}{
		{"wrong origin", "https://evil.example", 5, 2, u.ID, false},
		{"no user verification", ctrl.passkeys.origin, 1, 2, u.ID, false},
		{"invalid signature", ctrl.passkeys.origin, 5, 2, u.ID, true},
		{"wrong user", ctrl.passkeys.origin, 5, 2, uuid.New(), false},
		{"cloned counter", ctrl.passkeys.origin, 5, 1, u.ID, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ch, ck := begin(false)
			require.Error(t, ctrl.HandlePasskeyFinish(false)(httptest.NewRecorder(), request(assertion(ch, tc.origin, tc.flags, tc.counter, tc.handle, tc.tamper), false, ck)))
		})
	}
	t.Run("changed session and expiry", func(t *testing.T) {
		ch, ck := begin(true)
		r := request(registration(ch), true, ck)
		r.Header.Del("Cookie")
		r.AddCookie(ck)
		r.AddCookie(&http.Cookie{Name: cookieNameToken, Value: "other-session"})
		require.Error(t, ctrl.HandlePasskeyFinish(true)(httptest.NewRecorder(), r))
		ch, ck = begin(false)
		c := ctrl.passkeys.pending[ck.Value]
		c.expires = time.Now().Add(-time.Second)
		ctrl.passkeys.pending[ck.Value] = c
		require.Error(t, ctrl.HandlePasskeyFinish(false)(httptest.NewRecorder(), request(assertion(ch, ctrl.passkeys.origin, 5, 2, u.ID, false), false, ck)))
	})
	t.Run("list omits credential material", func(t *testing.T) {
		r := request(nil, true, nil)
		r.Method = "GET"
		w := httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePasskeyList()(w, r))
		require.NotContains(t, w.Body.String(), "credential")
		require.Contains(t, w.Body.String(), "My device")
	})
	require.Error(t, repos.Users.DeletePasskey(ctx, uuid.New(), stored.ID), "cross-user deletion")
	ch, ck := begin(false)
	route := chi.NewRouteContext()
	route.URLParams.Add("id", stored.ID.String())
	r := request(nil, true, nil).WithContext(context.WithValue(authed, chi.RouteCtxKey, route))
	r.Method = "DELETE"
	require.NoError(t, ctrl.HandlePasskeyDelete()(httptest.NewRecorder(), r))
	require.Error(t, ctrl.HandlePasskeyFinish(false)(httptest.NewRecorder(), request(assertion(ch, ctrl.passkeys.origin, 5, 2, u.ID, false), false, ck)), "deleted keys cannot log in")
	_, err = svc.User.Login(ctx, u.Email, "correct-password", false)
	require.NoError(t, err, "password fallback remains usable")
}

func TestPasskeyOriginConfig(t *testing.T) {
	for _, origin := range []string{"", "http://app.example.com", "https://app.example.com/path", "https://user:pass@app.example.com", "https://app.example.com?bad=1"} {
		_, err := newPasskeyRuntime(origin)
		require.Error(t, err)
	}
	for _, origin := range []string{"https://app.example.com", "http://localhost:3000"} {
		_, err := newPasskeyRuntime(origin)
		require.NoError(t, err)
	}
}
