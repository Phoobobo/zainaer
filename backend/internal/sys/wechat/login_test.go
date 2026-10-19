// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package wechat

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestWeChatExchange(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		ok         bool
	}{
		{"success", `{"openid":"verified","session_key":"never-return"}`, 200, true},
		{"invalid code", `{"errcode":40029,"errmsg":"secret"}`, 200, false},
		{"missing id", `{"session_key":"secret"}`, 200, false},
		{"bad json", `<html>secret</html>`, 200, false},
		{"redirect", ``, 302, false},
		{"server failure", `{"openid":"untrusted"}`, 500, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, "https", r.URL.Scheme)
				require.Equal(t, "api.weixin.qq.com", r.URL.Host)
				require.Equal(t, "app", r.URL.Query().Get("appid"))
				require.Equal(t, "code", r.URL.Query().Get("js_code"))
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})}
			id, err := Exchange(context.Background(), client, "app", "secret", "code")
			if tc.ok {
				require.NoError(t, err)
				require.Equal(t, "verified", id)
			} else {
				require.ErrorIs(t, err, ErrExchange)
				require.Empty(t, id)
			}
		})
	}
	client := &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return nil, errors.New("secret URL") })}
	_, err := Exchange(context.Background(), client, "app", "secret", "code")
	require.Equal(t, ErrExchange, err)
	_, err = Exchange(context.Background(), client, "app", "secret", "")
	require.Equal(t, ErrExchange, err)
}
