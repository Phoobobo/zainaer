// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package wechat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrExchange = errors.New("wechat login verification failed")

// Exchange never propagates URL-bearing transport errors or provider responses:
// both can contain the application secret, login code or session key.
func Exchange(ctx context.Context, client *http.Client, appID, secret, code string) (string, error) {
	if appID == "" || secret == "" || strings.TrimSpace(code) == "" || len(code) > 512 {
		return "", ErrExchange
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	query := url.Values{"appid": {appID}, "secret": {secret}, "js_code": {code}, "grant_type": {"authorization_code"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.weixin.qq.com/sns/jscode2session?"+query.Encode(), nil)
	if err != nil {
		return "", ErrExchange
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	safeClient := *client
	safeClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := safeClient.Do(req)
	if err != nil {
		return "", ErrExchange
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", ErrExchange
	}
	var result struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&result) != nil || result.ErrCode != 0 || result.OpenID == "" || len(result.OpenID) > 128 {
		return "", ErrExchange
	}
	return result.OpenID, nil
}
