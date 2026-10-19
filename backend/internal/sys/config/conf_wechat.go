// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package config

import "encoding/json"

// WeChatConf is disabled until the operator has registered a mini program.
type WeChatConf struct {
	Enabled   bool   `yaml:"enabled" conf:"default:false,env:WECHAT_ENABLED"`
	AppID     string `yaml:"app_id" conf:"env:WECHAT_APP_ID"`
	AppSecret string `yaml:"app_secret" conf:"env:WECHAT_APP_SECRET"`
}

func (c WeChatConf) MarshalJSON() ([]byte, error) {
	type plain WeChatConf
	value := plain(c)
	if value.AppSecret != "" {
		value.AppSecret = redactedValue
	}
	return json.Marshal(value)
}
