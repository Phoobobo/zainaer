// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/pkgs/hasher"
)

func TestRegistrationCollectionOnboarding(t *testing.T) {
	ctx := context.Background()
	owner, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "Owner", Email: fk.Email(), Password: "test-password"})
	require.NoError(t, err)
	groups, err := tRepos.Groups.GetAllGroups(ctx, owner.ID)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, "我的家", groups[0].Name)
	require.Equal(t, owner.DefaultGroupID, groups[0].ID)

	raw := uuid.NewString()
	_, err = tRepos.Groups.InvitationCreate(ctx, owner.DefaultGroupID, repo.GroupInvitationCreate{
		Token: hasher.HashToken(raw), ExpiresAt: time.Now().Add(time.Hour), Uses: 2,
	})
	require.NoError(t, err)
	invited, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "Invited", Email: fk.Email(), Password: "test-password", GroupToken: raw})
	require.NoError(t, err)
	require.Equal(t, owner.DefaultGroupID, invited.DefaultGroupID)
	groups, err = tRepos.Groups.GetAllGroups(ctx, invited.ID)
	require.NoError(t, err)
	require.Len(t, groups, 1, "invitation registration must not create a personal collection")
	require.Equal(t, owner.DefaultGroupID, groups[0].ID)

	existing, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "Existing", Email: fk.Email(), Password: "test-password"})
	require.NoError(t, err)
	joined, err := tRepos.Groups.InvitationAccept(ctx, hasher.HashToken(raw), existing.ID)
	require.NoError(t, err)
	require.Equal(t, owner.DefaultGroupID, joined.ID)
	groups, err = tRepos.Groups.GetAllGroups(ctx, existing.ID)
	require.NoError(t, err)
	require.Len(t, groups, 2, "joining must retain the existing user's personal collection")
}
