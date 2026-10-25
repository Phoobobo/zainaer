// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package services

import (
	"context"
	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/pkgs/hasher"
)

// VerifyPasskeyPassword performs step-up verification without creating a session.
func (svc *UserService) VerifyPasskeyPassword(ctx context.Context, uid uuid.UUID, password string) error {
	u, err := svc.repos.Users.GetOneID(ctx, uid)
	if err != nil || u.PasswordHash == "" || len(password) > 1024 {
		return ErrorInvalidLogin
	}
	match, _ := hasher.CheckPasswordHashCtx(ctx, password, u.PasswordHash)
	if !match {
		return ErrorInvalidLogin
	}
	return nil
}

// LoginPasskey is called only after a WebAuthn assertion has been verified and persisted.
func (svc *UserService) LoginPasskey(ctx context.Context, uid uuid.UUID) (UserAuthTokenDetail, error) {
	if _, err := svc.repos.Users.GetOneID(ctx, uid); err != nil {
		return UserAuthTokenDetail{}, ErrorInvalidLogin
	}
	return svc.createSessionToken(ctx, uid, true)
}
