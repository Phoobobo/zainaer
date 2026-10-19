// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package services

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWeChatIdentityAndBinding(t *testing.T) {
	ctx := context.Background()
	uid := uuid.NewString()
	key := WeChatIdentity("app", uid)
	require.NotEqual(t, key, WeChatIdentity("other-app", uid))
	_, err := tSvc.User.LoginWeChat(ctx, key, false)
	require.ErrorIs(t, err, ErrWeChatUnbound)
	first := newTestUserWithPassword(t, "password-1")
	second := newTestUserWithPassword(t, "password-2")
	require.NoError(t, tSvc.User.BindWeChat(ctx, first.ID, key))
	require.NoError(t, tSvc.User.BindWeChat(ctx, first.ID, key))
	require.Error(t, tSvc.User.BindWeChat(ctx, second.ID, key))
	require.Error(t, tSvc.User.BindWeChat(ctx, first.ID, WeChatIdentity("app", uuid.NewString())))
	token, err := tSvc.User.LoginWeChat(ctx, key, false)
	require.NoError(t, err)
	actual, err := tSvc.User.GetSelf(ctx, token.Raw)
	require.NoError(t, err)
	require.Equal(t, first.ID, actual.ID)
	require.Equal(t, first.DefaultGroupID, actual.DefaultGroupID)
	_, err = tSvc.User.Login(ctx, first.Email, "password-1", false)
	require.NoError(t, err)
}
func TestWeChatRegistration(t *testing.T) {
	ctx := context.Background()
	key := WeChatIdentity("app", uuid.NewString())
	token, err := tSvc.User.LoginWeChat(ctx, key, true)
	require.NoError(t, err)
	first, err := tSvc.User.GetSelf(ctx, token.Raw)
	require.NoError(t, err)
	require.False(t, first.IsSuperuser)
	again, err := tSvc.User.LoginWeChat(ctx, key, false)
	require.NoError(t, err)
	second, err := tSvc.User.GetSelf(ctx, again.Raw)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	_, err = tSvc.User.Login(ctx, first.Email, "", false)
	require.Error(t, err)
}
