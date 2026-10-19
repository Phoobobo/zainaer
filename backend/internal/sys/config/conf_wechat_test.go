// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package config

import (
	"encoding/json"
	"github.com/ardanlabs/conf/v3"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWeChatSecretRedacted(t *testing.T) {
	b, err := json.Marshal(Config{WeChat: WeChatConf{Enabled: true, AppID: "app", AppSecret: "secret-never-log"}})
	require.NoError(t, err)
	require.NotContains(t, string(b), "secret-never-log")
	require.Contains(t, string(b), redactedValue)
}

func TestWeChatEnvironmentNames(t *testing.T) {
	t.Setenv("HBOX_WECHAT_ENABLED", "true")
	t.Setenv("HBOX_WECHAT_APP_ID", "test-app")
	t.Setenv("HBOX_WECHAT_APP_SECRET", "test-secret")
	var cfg Config
	_, err := conf.Parse("HBOX", &cfg)
	require.NoError(t, err)
	require.True(t, cfg.WeChat.Enabled)
	require.Equal(t, "test-app", cfg.WeChat.AppID)
	require.Equal(t, "test-secret", cfg.WeChat.AppSecret)
}
