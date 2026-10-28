// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
)

func TestLoginOIDCStillLinksExistingUserByEmail(t *testing.T) {
	ctx := context.Background()
	email := fk.Email()
	local, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "Local", Email: email, Password: "test-password"})
	require.NoError(t, err)

	token, err := tSvc.User.LoginOIDC(ctx, "https://issuer.example", "oidc-subject-1", email, "Local")
	require.NoError(t, err)
	require.NotEmpty(t, token.Raw)

	got, err := tRepos.Users.GetOneID(ctx, local.ID)
	require.NoError(t, err)
	require.NotNil(t, got.OidcIssuer)
	require.Equal(t, "https://issuer.example", *got.OidcIssuer)
	require.Equal(t, "oidc-subject-1", *got.OidcSubject)
}

func TestLoginPhooboboDoesNotLinkUnverifiedOrSyntheticEmail(t *testing.T) {
	ctx := context.Background()
	issuer := "https://account.phoobobo.top"

	unverifiedEmail := fk.Email()
	unverified, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "Unverified", Email: unverifiedEmail, Password: "test-password"})
	require.NoError(t, err)
	_, err = tSvc.User.LoginPhoobobo(ctx, PhooboboLogin{
		Issuer: issuer, Subject: "phoobobo:unverified", Email: unverifiedEmail, CreateEmail: unverifiedEmail, Name: "Unverified",
	})
	require.ErrorIs(t, err, ErrPhooboboEmailUnverified)
	got, err := tRepos.Users.GetOneID(ctx, unverified.ID)
	require.NoError(t, err)
	require.Nil(t, got.OidcIssuer)
	require.Nil(t, got.OidcSubject)
	_, err = tRepos.Users.GetOneOIDC(ctx, issuer, "phoobobo:unverified")
	require.True(t, ent.IsNotFound(err))

	verifiedEmail := fk.Email()
	verified, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "Verified", Email: verifiedEmail, Password: "test-password"})
	require.NoError(t, err)
	token, err := tSvc.User.LoginPhoobobo(ctx, PhooboboLogin{
		Issuer: issuer, Subject: "phoobobo:verified", Email: verifiedEmail, CreateEmail: verifiedEmail,
		Name: "Verified", EmailVerified: true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, token.Raw)
	got, err = tRepos.Users.GetOneID(ctx, verified.ID)
	require.NoError(t, err)
	require.Equal(t, "phoobobo:verified", *got.OidcSubject)

	knownEmail := fk.Email()
	token, err = tSvc.User.LoginPhoobobo(ctx, PhooboboLogin{
		Issuer: issuer, Subject: "phoobobo:known", Email: knownEmail, CreateEmail: knownEmail, Name: "Known",
	})
	require.NoError(t, err)
	require.NotEmpty(t, token.Raw)
	otherEmail := fk.Email()
	other, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "Other", Email: otherEmail, Password: "test-password"})
	require.NoError(t, err)
	token, err = tSvc.User.LoginPhoobobo(ctx, PhooboboLogin{
		Issuer: issuer, Subject: "phoobobo:known", Email: otherEmail, CreateEmail: otherEmail, Name: "Known",
	})
	require.NoError(t, err)
	require.NotEmpty(t, token.Raw)
	got, err = tRepos.Users.GetOneID(ctx, other.ID)
	require.NoError(t, err)
	require.Nil(t, got.OidcSubject)

	phoneEmail := PhooboboSyntheticEmail("13900001111")
	phoneUser, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "Phone", Email: phoneEmail, Password: "test-password"})
	require.NoError(t, err)
	_, err = tSvc.User.LoginPhoobobo(ctx, PhooboboLogin{
		Issuer: issuer, Subject: "phoobobo:phone", CreateEmail: phoneEmail, Name: "Phone", EmailVerified: true,
	})
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrPhooboboEmailUnverified)
	got, err = tRepos.Users.GetOneID(ctx, phoneUser.ID)
	require.NoError(t, err)
	require.Nil(t, got.OidcIssuer)
	require.Nil(t, got.OidcSubject)

	freshPhone := PhooboboSyntheticEmail("13900002222")
	token, err = tSvc.User.LoginPhoobobo(ctx, PhooboboLogin{
		Issuer: issuer, Subject: "phoobobo:fresh-phone", CreateEmail: freshPhone, Name: "新手机",
	})
	require.NoError(t, err)
	require.NotEmpty(t, token.Raw)
	created, err := tRepos.Users.GetOneEmail(ctx, freshPhone)
	require.NoError(t, err)
	require.Equal(t, "phoobobo:fresh-phone", *created.OidcSubject)
}
