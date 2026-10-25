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
