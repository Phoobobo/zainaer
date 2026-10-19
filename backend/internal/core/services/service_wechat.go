// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

var ErrWeChatUnbound = errors.New("wechat account is not linked")

// WeChatIdentity scopes OpenID to its application, without storing the raw ID.
func WeChatIdentity(appID, openID string) string {
	hash := sha256.Sum256([]byte(appID + "\x00" + openID))
	return hex.EncodeToString(hash[:])
}

func (svc *UserService) LoginWeChat(ctx context.Context, identity string, register bool) (UserAuthTokenDetail, error) {
	if identity == "" {
		return UserAuthTokenDetail{}, ErrorInvalidLogin
	}
	usr, err := svc.repos.Users.GetWeChat(ctx, identity)
	if ent.IsNotFound(err) {
		if !register {
			return UserAuthTokenDetail{}, ErrWeChatUnbound
		}
		// This reserved address is never used to match/link an existing account.
		usr, err = svc.registerExternalUser(ctx, "wx-"+identity+"@wechat.invalid", "微信用户", func(in repo.UserCreate) (repo.UserOut, error) {
			return svc.repos.Users.CreateWeChat(ctx, in, identity)
		})
		if ent.IsConstraintError(err) {
			usr, err = svc.repos.Users.GetWeChat(ctx, identity)
		}
	}
	if err != nil {
		return UserAuthTokenDetail{}, err
	}
	return svc.createSessionToken(ctx, usr.ID, true)
}

func (svc *UserService) BindWeChat(ctx context.Context, uid uuid.UUID, identity string) error {
	if uid == uuid.Nil || identity == "" {
		return ErrorInvalidLogin
	}
	return svc.repos.Users.BindWeChat(ctx, uid, identity)
}
