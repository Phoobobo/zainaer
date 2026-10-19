// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package v1

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

type wechatTransport func(*http.Request) (*http.Response, error)

func (f wechatTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestWeChatHandler(t *testing.T) {
	dir := t.TempDir()
	db, err := ent.Open("sqlite3", "file:"+filepath.Join(dir, "test.db")+"?_fk=1&_time_format=sqlite")
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.Schema.Create(context.Background()))
	repos := repo.New(db, nil, config.Storage{PrefixPath: dir, ConnString: "file://" + dir}, "mem://{{ .Topic }}", config.Thumbnail{})
	ctrl := &V1Controller{repo: repos, svc: services.New(repos), config: &config.Config{}}
	req := func(body string) (*httptest.ResponseRecorder, error) {
		r := httptest.NewRequest("POST", "/api/v1/users/login/wechat", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		err := ctrl.HandleWeChatLogin(false)(w, r)
		return w, err
	}
	_, err = req(`{"code":"test"}`)
	require.Error(t, err)
	ctrl.config.WeChat = config.WeChatConf{Enabled: true, AppID: "app", AppSecret: "secret"}
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original }()
	calls := 0
	http.DefaultTransport = wechatTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"openid":"trusted-id","session_key":"private-session-key"}`)), Header: make(http.Header)}, nil
	})
	_, err = req(`{"openid":"forged"}`)
	require.Error(t, err)
	require.Zero(t, calls)
	w, err := req(`{"code":"test","register":true}`)
	require.NoError(t, err)
	require.Equal(t, 409, w.Code)
	require.Contains(t, w.Body.String(), `"registrationAllowed":false`)
	n, err := db.User.Query().Count(context.Background())
	require.NoError(t, err)
	require.Zero(t, n)
	ctrl.allowRegistration = true
	w, err = req(`{"code":"test","register":true,"openid":"forged"}`)
	require.NoError(t, err)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "Bearer ")
	require.NotContains(t, w.Body.String(), "trusted-id")
	require.NotContains(t, w.Body.String(), "private-session-key")
	require.Empty(t, w.Header().Values("Set-Cookie"))
	_, err = repos.Users.GetWeChat(context.Background(), services.WeChatIdentity("app", "trusted-id"))
	require.NoError(t, err)
	ctrl.allowRegistration = false
	w, err = req(`{"code":"next"}`)
	require.NoError(t, err)
	require.Equal(t, 200, w.Code)
}
