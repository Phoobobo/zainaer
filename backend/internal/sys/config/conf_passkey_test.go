// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package config

import (
	"github.com/ardanlabs/conf/v3"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPasskeyEnvironment(t *testing.T) {
	t.Setenv("HBOX_PASSKEY_ORIGIN", "https://app.example.test")
	var cfg Config
	_, err := conf.Parse("HBOX", &cfg)
	require.NoError(t, err)
	require.Equal(t, "https://app.example.test", cfg.Passkey.Origin)
}

func TestPhooboboAccountEnvironment(t *testing.T) {
	t.Setenv("HBOX_PHOOBOBO_ACCOUNT_URL", "https://account.phoobobo.top")
	t.Setenv("HBOX_PHOOBOBO_ACCOUNT_INTERNAL", "http://127.0.0.1:7750")
	var cfg Config
	_, err := conf.Parse("HBOX", &cfg)
	require.NoError(t, err)
	require.Equal(t, "https://account.phoobobo.top", cfg.Phoobobo.AccountURL)
	require.Equal(t, "http://127.0.0.1:7750", cfg.Phoobobo.AccountInternal)
	require.True(t, cfg.Phoobobo.Enabled())
}

func TestPhooboboAccountDisabledWhenUnset(t *testing.T) {
	t.Setenv("HBOX_PHOOBOBO_ACCOUNT_URL", "")
	t.Setenv("HBOX_PHOOBOBO_ACCOUNT_INTERNAL", "")
	var cfg Config
	_, err := conf.Parse("HBOX", &cfg)
	require.NoError(t, err)
	require.Empty(t, cfg.Phoobobo.AccountURL)
	require.False(t, cfg.Phoobobo.Enabled())
	cfg.Phoobobo.AccountURL = "  "
	require.False(t, cfg.Phoobobo.Enabled())
}
