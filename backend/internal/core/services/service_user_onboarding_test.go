// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/usergroup"
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
	require.Equal(t, "Owner的家", groups[0].Name)
	require.NotNil(t, owner.CollectionNamingID)
	require.Equal(t, owner.DefaultGroupID, *owner.CollectionNamingID)
	require.Equal(t, owner.DefaultGroupID, groups[0].ID)

	raw := uuid.NewString()
	_, err = tRepos.Groups.InvitationCreate(ctx, owner.DefaultGroupID, repo.GroupInvitationCreate{
		Token: hasher.HashToken(raw), ExpiresAt: time.Now().Add(time.Hour), Uses: 2,
	})
	require.NoError(t, err)
	invited, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "Invited", Email: fk.Email(), Password: "test-password", GroupToken: raw})
	require.NoError(t, err)
	require.Equal(t, owner.DefaultGroupID, invited.DefaultGroupID)
	require.Nil(t, invited.CollectionNamingID)
	require.ErrorIs(t, tRepos.Users.CompleteCollectionNaming(ctx, invited.ID, "Not allowed"), repo.ErrCollectionNamingUnavailable)
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

func TestCollectionNamingCompletion(t *testing.T) {
	ctx := context.Background()
	usr, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "小明", Email: fk.Email(), Password: "test-password"})
	require.NoError(t, err)
	require.ErrorIs(t, tRepos.Users.CompleteCollectionNaming(ctx, usr.ID, "  "), repo.ErrCollectionNameInvalid)
	require.ErrorIs(t, tRepos.Users.CompleteCollectionNaming(ctx, usr.ID, strings.Repeat("家", 100)), repo.ErrCollectionNameInvalid)
	fresh, err := tRepos.Users.GetOneID(ctx, usr.ID)
	require.NoError(t, err)
	require.NotNil(t, fresh.CollectionNamingID)
	require.NoError(t, tRepos.Users.CompleteCollectionNaming(ctx, usr.ID, "  小明和小红的家  "))
	fresh, err = tRepos.Users.GetOneID(ctx, usr.ID)
	require.NoError(t, err)
	require.Nil(t, fresh.CollectionNamingID)
	group, err := tRepos.Groups.GroupByID(ctx, usr.DefaultGroupID)
	require.NoError(t, err)
	require.Equal(t, "小明和小红的家", group.Name)
	require.ErrorIs(t, tRepos.Users.CompleteCollectionNaming(ctx, usr.ID, "second tab"), repo.ErrCollectionNamingUnavailable)
}

func TestDefaultCollectionName(t *testing.T) {
	require.Equal(t, "小明的家", defaultCollectionName(" 小明 "))
	require.LessOrEqual(t, len(defaultCollectionName(strings.Repeat("明", 255))), 255)
}

func TestCollectionNamingCannotRenameAnotherCollection(t *testing.T) {
	ctx := context.Background()
	usr, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "Owner", Email: fk.Email(), Password: "test-password"})
	require.NoError(t, err)
	// Changing the preferred/default collection must not change the naming target.
	require.NoError(t, tRepos.Users.UpdateDefaultGroup(ctx, usr.ID, tGroup.ID))
	require.NoError(t, tRepos.Users.CompleteCollectionNaming(ctx, usr.ID, "Initial home"))
	group, err := tRepos.Groups.GroupByID(ctx, usr.DefaultGroupID)
	require.NoError(t, err)
	require.Equal(t, "Initial home", group.Name)
	other, err := tRepos.Groups.GroupByID(ctx, tGroup.ID)
	require.NoError(t, err)
	require.Equal(t, tGroup.Name, other.Name)
}

func TestCollectionNamingRequiresCurrentOwnership(t *testing.T) {
	ctx := context.Background()
	usr, err := tSvc.User.RegisterUser(ctx, UserRegistration{Name: "Owner", Email: fk.Email(), Password: "test-password"})
	require.NoError(t, err)
	_, err = tClient.UserGroup.Update().Where(usergroup.UserID(usr.ID), usergroup.GroupID(usr.DefaultGroupID)).SetRole(usergroup.RoleUser).Save(ctx)
	require.NoError(t, err)
	require.ErrorIs(t, tRepos.Users.CompleteCollectionNaming(ctx, usr.ID, "Denied"), repo.ErrCollectionNamingUnavailable)
}

func TestOIDCCollectionNaming(t *testing.T) {
	usr, err := tSvc.User.registerOIDCUser(context.Background(), "https://issuer.example", uuid.NewString(), fk.Email(), "小红")
	require.NoError(t, err)
	require.NotNil(t, usr.CollectionNamingID)
	group, err := tRepos.Groups.GroupByID(context.Background(), usr.DefaultGroupID)
	require.NoError(t, err)
	require.Equal(t, "小红的家", group.Name)
	require.Nil(t, tUser.CollectionNamingID, "existing users are not enrolled automatically")
}
