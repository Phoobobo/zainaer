// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"

	_ "github.com/sysadminsmedia/homebox/backend/pkgs/cgofreesqlite"
)

func TestPhooboboAccountLogin(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := ent.Open("sqlite3", "file:"+filepath.Join(dir, "phoobobo.db")+"?_fk=1&_time_format=sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.Schema.Create(ctx))
	repos := repo.New(db, nil, config.Storage{PrefixPath: dir, ConnString: "file://" + dir}, "mem://{{ .Topic }}", config.Thumbnail{})
	svc := services.New(repos, services.WithAutoIncrementAssetID(false))

	type seen struct {
		host    string
		cookie  string
		path    string
		follows int
	}
	newCtrl := func(t *testing.T, public string, handler http.HandlerFunc) (*V1Controller, *httptest.Server, *seen) {
		t.Helper()
		got := &seen{}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/followed" {
				got.follows++
				http.NotFound(w, r)
				return
			}
			got.host = r.Host
			got.cookie = r.Header.Get("Cookie")
			got.path = r.URL.Path
			handler(w, r)
		}))
		t.Cleanup(srv.Close)
		ctrl := &V1Controller{
			repo: repos,
			svc:  svc,
			config: &config.Config{
				Options: config.Options{Hostname: "https://zainaer.phoobobo.top"},
				Phoobobo: config.PhooboboConf{
					AccountURL:      public,
					AccountInternal: srv.URL,
				},
			},
		}
		return ctrl, srv, got
	}
	login := func(cookies ...*http.Cookie) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "https://zainaer.phoobobo.top/api/v1/users/login/phoobobo", nil)
		r.Host = "zainaer.phoobobo.top"
		for _, c := range cookies {
			r.AddCookie(c)
		}
		return r
	}
	session := func(fields map[string]any) []byte {
		t.Helper()
		if _, ok := fields["issuer"]; !ok {
			fields["issuer"] = phooboboAccountIssuer
		}
		b, err := json.Marshal(fields)
		require.NoError(t, err)
		return b
	}
	writeJSON := func(w http.ResponseWriter, status int, body []byte) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}
	assertLocalLogin := func(t *testing.T, w *httptest.ResponseRecorder) {
		t.Helper()
		require.Equal(t, http.StatusFound, w.Code)
		require.Equal(t, "/", w.Header().Get("Location"))
		require.Empty(t, w.Result().Cookies())
	}
	assertAccountLogin := func(t *testing.T, w *httptest.ResponseRecorder) {
		t.Helper()
		require.Equal(t, http.StatusFound, w.Code)
		loc, err := url.Parse(w.Header().Get("Location"))
		require.NoError(t, err)
		require.Equal(t, "https", loc.Scheme)
		require.Equal(t, phooboboAccountHost, loc.Host)
		require.Equal(t, "/login", loc.Path)
		require.Equal(t, "login", loc.Query().Get("mode"))
		require.Equal(t, "https://zainaer.phoobobo.top/api/v1/users/login/phoobobo", loc.Query().Get("next"))
		require.Empty(t, w.Result().Cookies())
	}

	t.Run("200 stores the fixed issuer and sends the account host", func(t *testing.T) {
		ctrl, _, got := newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, session(map[string]any{
				"id": 7, "subject": "phoobobo:7", "display_name": "Ada", "email": "ada@example.test", "guest": false,
			}))
		})
		w := httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(&http.Cookie{Name: phooboboCookie, Value: "sid-ada"})))
		require.Equal(t, http.StatusFound, w.Code)
		require.Equal(t, "/home", w.Header().Get("Location"))
		require.Equal(t, phooboboAccountHost, got.host)
		require.Equal(t, "/api/account/session", got.path)
		require.Equal(t, "phoobobo_sid=sid-ada", got.cookie)
		var token string
		for _, c := range w.Result().Cookies() {
			if c.Name == cookieNameToken {
				token = c.Value
			}
		}
		require.NotEmpty(t, token)
		usr, err := repos.Users.GetOneEmail(ctx, "ada@example.test")
		require.NoError(t, err)
		require.Equal(t, phooboboAccountIssuer, *usr.OidcIssuer)
		require.Equal(t, "phoobobo:7", *usr.OidcSubject)
		require.Equal(t, "Ada", usr.Name)
	})

	t.Run("401 redirects to account login with next", func(t *testing.T) {
		ctrl, _, got := newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusUnauthorized, []byte(`{"error":"not signed in"}`))
		})
		w := httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(&http.Cookie{Name: phooboboCookie, Value: "expired"})))
		assertAccountLogin(t, w)
		require.Equal(t, phooboboAccountHost, got.host)
		require.Equal(t, "phoobobo_sid=expired", got.cookie)
	})

	t.Run("issuer is fixed and a different issuer does not sign in", func(t *testing.T) {
		ctrl, _, _ := newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, session(map[string]any{
				"id": 9, "subject": "phoobobo:9", "display_name": "Eve", "email": "eve@example.test",
				"issuer": "https://bbs.zjsy.phoobobo.top",
			}))
		})
		w := httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(&http.Cookie{Name: phooboboCookie, Value: "sid-eve"})))
		assertLocalLogin(t, w)
		_, err := repos.Users.GetOneEmail(ctx, "eve@example.test")
		require.True(t, ent.IsNotFound(err))

		ctrl, _, _ = newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, session(map[string]any{
				"id": 8, "subject": "phoobobo:8", "display_name": "Bea", "email": "bea@example.test",
				"issuer": phooboboAccountIssuer + "/",
			}))
		})
		w = httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(&http.Cookie{Name: phooboboCookie, Value: "sid-bea"})))
		require.Equal(t, "/home", w.Header().Get("Location"))
		usr, err := repos.Users.GetOneEmail(ctx, "bea@example.test")
		require.NoError(t, err)
		require.Equal(t, phooboboAccountIssuer, *usr.OidcIssuer)
		require.Equal(t, "phoobobo:8", *usr.OidcSubject)
	})

	t.Run("guest and anonymous sessions are not signed in", func(t *testing.T) {
		for _, body := range []map[string]any{
			{"id": 3, "subject": "phoobobo:3", "display_name": "Guest", "email": "guest@example.test", "guest": true},
			{"id": 4, "subject": "phoobobo:4", "display_name": "Anon", "email": "anon@example.test", "anonymous": true},
			{"id": 0, "subject": "", "display_name": "", "email": ""},
		} {
			ctrl, _, _ := newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, session(body))
			})
			w := httptest.NewRecorder()
			require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(&http.Cookie{Name: phooboboCookie, Value: "guest"})))
			assertAccountLogin(t, w)
		}
		_, err := repos.Users.GetOneEmail(ctx, "guest@example.test")
		require.True(t, ent.IsNotFound(err))
	})

	t.Run("legacy zjsy_sid is forwarded when phoobobo_sid is absent", func(t *testing.T) {
		ctrl, _, got := newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, session(map[string]any{
				"id": 11, "subject": "phoobobo:11", "display_name": "Legacy", "email": "legacy@example.test",
			}))
		})
		w := httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(&http.Cookie{Name: phooboboLegacyCookie, Value: "old-sid"})))
		require.Equal(t, "/home", w.Header().Get("Location"))
		require.Equal(t, "zjsy_sid=old-sid", got.cookie)
		require.NotContains(t, got.cookie, phooboboCookie)
	})

	t.Run("phoobobo_sid wins and a signed-out new cookie falls back to zjsy_sid", func(t *testing.T) {
		var cookies []string
		ctrl, _, _ := newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {
			cookies = append(cookies, r.Header.Get("Cookie"))
			if strings.HasPrefix(r.Header.Get("Cookie"), phooboboCookie+"=") {
				writeJSON(w, http.StatusUnauthorized, []byte(`{"error":"not signed in"}`))
				return
			}
			writeJSON(w, http.StatusOK, session(map[string]any{
				"id": 12, "subject": "phoobobo:12", "display_name": "Both", "email": "both@example.test",
			}))
		})
		w := httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(
			&http.Cookie{Name: phooboboCookie, Value: "new-empty"},
			&http.Cookie{Name: phooboboLegacyCookie, Value: "still-good"},
		)))
		require.Equal(t, "/home", w.Header().Get("Location"))
		require.Equal(t, []string{"phoobobo_sid=new-empty", "zjsy_sid=still-good"}, cookies)
	})

	t.Run("no cookie skips the account service and redirects to login", func(t *testing.T) {
		hits := 0
		ctrl, _, _ := newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {
			hits++
		})
		w := httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login()))
		assertAccountLogin(t, w)
		require.Zero(t, hits)
	})

	t.Run("phone-only accounts use a stable synthetic email", func(t *testing.T) {
		ctrl, _, _ := newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, session(map[string]any{
				"id": 15, "display_name": "手机用户", "email": "", "phone": "13800138000",
			}))
		})
		w := httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(&http.Cookie{Name: phooboboCookie, Value: "sid-phone"})))
		require.Equal(t, "/home", w.Header().Get("Location"))
		usr, err := repos.Users.GetOneEmail(ctx, "13800138000@users.phoobobo.top")
		require.NoError(t, err)
		require.Equal(t, "phoobobo:15", *usr.OidcSubject)
		require.Equal(t, phooboboAccountIssuer, *usr.OidcIssuer)
	})

	t.Run("account outage and bad payloads return to local login", func(t *testing.T) {
		ctrl, srv, _ := newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {})
		srv.Close()
		w := httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(&http.Cookie{Name: phooboboCookie, Value: "sid"})))
		assertLocalLogin(t, w)

		ctrl, _, got := newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/followed", http.StatusFound)
		})
		w = httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(&http.Cookie{Name: phooboboCookie, Value: "sid"})))
		assertLocalLogin(t, w)
		require.Zero(t, got.follows)

		ctrl, _, _ = newCtrl(t, phooboboAccountIssuer, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, []byte("{"))
		})
		w = httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(&http.Cookie{Name: phooboboCookie, Value: "sid"})))
		assertLocalLogin(t, w)
	})

	t.Run("public url is called without overriding host", func(t *testing.T) {
		var host string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host = r.Host
			writeJSON(w, http.StatusOK, session(map[string]any{
				"id": 21, "subject": "phoobobo:21", "display_name": "Public", "email": "public@example.test",
			}))
		}))
		t.Cleanup(srv.Close)
		ctrl := &V1Controller{
			repo: repos,
			svc:  svc,
			config: &config.Config{
				Options:  config.Options{Hostname: "https://zainaer.phoobobo.top"},
				Phoobobo: config.PhooboboConf{AccountURL: srv.URL},
			},
		}
		w := httptest.NewRecorder()
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, login(&http.Cookie{Name: phooboboCookie, Value: "sid-public"})))
		require.Equal(t, "/home", w.Header().Get("Location"))
		require.Equal(t, strings.TrimPrefix(srv.URL, "http://"), host)
		require.NotEqual(t, phooboboAccountHost, host)
		usr, err := repos.Users.GetOneOIDC(ctx, phooboboAccountIssuer, "phoobobo:21")
		require.NoError(t, err)
		require.Equal(t, "public@example.test", usr.Email)
	})
}

func TestPhooboboRegisterAndLogoutRedirects(t *testing.T) {
	ctrl := &V1Controller{config: &config.Config{
		Options:  config.Options{Hostname: "https://zainaer.phoobobo.top"},
		Phoobobo: config.PhooboboConf{AccountURL: phooboboAccountIssuer},
	}}
	loginRedirect := func(raw string) *url.URL {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, raw, nil)
		r.Host = "zainaer.phoobobo.top"
		require.NoError(t, ctrl.HandlePhooboboLogin()(w, r))
		require.Equal(t, http.StatusFound, w.Code)
		loc, err := url.Parse(w.Header().Get("Location"))
		require.NoError(t, err)
		return loc
	}
	registerURL := loginRedirect("https://zainaer.phoobobo.top/api/v1/users/login/phoobobo?mode=register")
	require.Equal(t, "/login", registerURL.Path)
	require.Equal(t, "register", registerURL.Query().Get("mode"))
	require.Equal(t, "https://zainaer.phoobobo.top/api/v1/users/login/phoobobo", registerURL.Query().Get("next"))
	require.Equal(t, "login", loginRedirect("https://zainaer.phoobobo.top/api/v1/users/login/phoobobo?mode=nope").Query().Get("mode"))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://zainaer.phoobobo.top/api/v1/users/logout/phoobobo", nil)
	r.Host = "zainaer.phoobobo.top"
	r.AddCookie(&http.Cookie{Name: cookieNameToken, Value: "local-session"})
	require.NoError(t, ctrl.HandlePhooboboLogout()(w, r))
	require.Equal(t, http.StatusFound, w.Code)
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "https", loc.Scheme)
	require.Equal(t, phooboboAccountHost, loc.Host)
	require.Equal(t, "/logout", loc.Path)
	require.Equal(t, "https://zainaer.phoobobo.top/", loc.Query().Get("next"))
	cleared := false
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieNameToken && c.Value == "" {
			cleared = true
		}
	}
	require.True(t, cleared)

	ctrl.config.Phoobobo.AccountURL = ""
	w = httptest.NewRecorder()
	require.NoError(t, ctrl.HandlePhooboboLogout()(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/logout/phoobobo", nil)))
	require.Equal(t, "/", w.Header().Get("Location"))
}

func TestPhooboboLoginDisabled(t *testing.T) {
	ctrl := &V1Controller{config: &config.Config{}}
	w := httptest.NewRecorder()
	err := ctrl.HandlePhooboboLogin()(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/login/phoobobo", nil))
	var reqErr *validate.RequestError
	require.ErrorAs(t, err, &reqErr)
	require.Equal(t, http.StatusForbidden, reqErr.Status)
}

func TestPhooboboCallbackURL(t *testing.T) {
	callback := func(hostname, host, proto string) string {
		ctrl := &V1Controller{config: &config.Config{Options: config.Options{Hostname: hostname}}}
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/login/phoobobo", nil)
		r.Host = host
		if proto != "" {
			r.Header.Set("X-Forwarded-Proto", proto)
		}
		return ctrl.phooboboCallbackURL(r)
	}
	require.Equal(t, "https://zainaer.phoobobo.top/api/v1/users/login/phoobobo", callback("https://zainaer.phoobobo.top", "127.0.0.1:7745", ""))
	require.Equal(t, "https://zainaer.phoobobo.top/api/v1/users/login/phoobobo", callback("zainaer.phoobobo.top", "127.0.0.1:7745", ""))
	require.Equal(t, "http://localhost:3000/api/v1/users/login/phoobobo", callback("localhost:3000", "ignored", ""))
	require.Equal(t, "https://zainaer.phoobobo.top/api/v1/users/login/phoobobo", callback("", "zainaer.phoobobo.top", ""))
	require.Equal(t, "http://127.0.0.1:7745/api/v1/users/login/phoobobo", callback("", "127.0.0.1:7745", ""))
	require.Equal(t, "https://127.0.0.1:7745/api/v1/users/login/phoobobo", callback("", "127.0.0.1:7745", "https"))
}

func TestPhooboboUpstreamHost(t *testing.T) {
	require.Equal(t, phooboboAccountHost, phooboboUpstreamHost("https://account.phoobobo.top"))
	require.Equal(t, phooboboAccountHost, phooboboUpstreamHost("https://account.phoobobo.top:443/"))
	require.Equal(t, phooboboAccountHost, phooboboUpstreamHost(""))
}
