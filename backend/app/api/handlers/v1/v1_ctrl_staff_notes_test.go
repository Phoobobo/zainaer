// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo

package v1

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
)

func staffNotesFixture(t *testing.T) (context.Context, *V1Controller, *repo.AllRepos, repo.Group, repo.UserOut, repo.UserOut, repo.EntityOut) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	db, err := ent.Open("sqlite3", "file:"+filepath.Join(dir, "staff-notes.db")+"?_fk=1&_time_format=sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.Schema.Create(ctx))

	repos := repo.New(db, nil, config.Storage{PrefixPath: dir, ConnString: "file://" + dir}, "mem://{{ .Topic }}", config.Thumbnail{})
	ctrl := &V1Controller{repo: repos, svc: services.New(repos, services.WithAutoIncrementAssetID(false))}

	g, err := repos.Groups.GroupCreate(ctx, "staff-notes", uuid.Nil)
	require.NoError(t, err)

	pw := "test-password"
	owner, err := repos.Users.Create(ctx, repo.UserCreate{
		Name: "Owner", Email: uuid.NewString() + "@example.com", Password: &pw, IsOwner: true, DefaultGroupID: g.ID,
	})
	require.NoError(t, err)
	member, err := repos.Users.Create(ctx, repo.UserCreate{
		Name: "Member", Email: uuid.NewString() + "@example.com", Password: &pw, IsOwner: false, DefaultGroupID: g.ID,
	})
	require.NoError(t, err)

	lt, err := repos.EntityTypes.GetDefault(ctx, g.ID, true)
	require.NoError(t, err)
	it, err := repos.EntityTypes.GetDefault(ctx, g.ID, false)
	require.NoError(t, err)
	loc, err := repos.Entities.Create(ctx, g.ID, repo.EntityCreate{Name: "Drawer", EntityTypeID: lt.ID})
	require.NoError(t, err)
	item, err := repos.Entities.Create(ctx, g.ID, repo.EntityCreate{Name: "Keys", EntityTypeID: it.ID, ParentID: loc.ID, Quantity: 1})
	require.NoError(t, err)

	return ctx, ctrl, repos, g, owner, member, item
}

func withTenant(ctx context.Context, user repo.UserOut, gid uuid.UUID) services.Context {
	auth := services.SetTenantCtx(services.SetUserCtx(ctx, &user, ""), gid)
	return services.NewContext(auth)
}

func TestPrivilegedStaffNotesVisibility(t *testing.T) {
	ctx, ctrl, repos, g, owner, member, item := staffNotesFixture(t)
	note := "purchase channel: WeChat"
	ownerCtx := withTenant(ctx, owner, g.ID)

	require.NoError(t, ctrl.authorizeStaffNotesWrite(ownerCtx, &repo.EntityUpdate{StaffNotes: &note}))
	updated, err := repos.Entities.UpdateByGroup(ctx, g.ID, repo.EntityUpdate{
		ID: item.ID, Name: item.Name, Description: item.Description, Quantity: item.Quantity,
		EntityTypeID: item.EntityType.ID, ParentID: item.Parent.ID, Notes: item.Notes, StaffNotes: &note,
	})
	require.NoError(t, err)

	visible, err := ctrl.filterEntityStaffNotes(ownerCtx, updated)
	require.NoError(t, err)
	require.NotNil(t, visible.StaffNotes)
	require.Equal(t, note, *visible.StaffNotes)
	require.NotNil(t, visible.PrivateToSelf)
	require.False(t, *visible.PrivateToSelf)

	memberCtx := withTenant(ctx, member, g.ID)
	hidden, err := ctrl.filterEntityStaffNotes(memberCtx, updated)
	require.NoError(t, err)
	require.Nil(t, hidden.StaffNotes)
	require.Nil(t, hidden.PrivateToSelf)

	payload, err := json.Marshal(hidden)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "staffNotes")
	require.NotContains(t, string(payload), "privateToSelf")
	require.NotContains(t, string(payload), note)

	err = ctrl.authorizeStaffNotesWrite(memberCtx, &repo.EntityUpdate{StaffNotes: &note})
	require.Error(t, err)
	var reqErr *validate.RequestError
	require.ErrorAs(t, err, &reqErr)
	require.Equal(t, 403, reqErr.Status)

	require.NoError(t, repos.Groups.SetMemberPrivileged(ctx, g.ID, member.ID, true))
	privCtx := withTenant(ctx, member, g.ID)
	shown, err := ctrl.filterEntityStaffNotes(privCtx, updated)
	require.NoError(t, err)
	require.NotNil(t, shown.StaffNotes)
	require.Equal(t, note, *shown.StaffNotes)
}

func TestPrivateToSelfHidesFromOthers(t *testing.T) {
	ctx, ctrl, repos, g, owner, member, item := staffNotesFixture(t)
	require.NoError(t, repos.Groups.SetMemberPrivileged(ctx, g.ID, member.ID, true))

	memberCtx := withTenant(ctx, member, g.ID)
	yes := true
	require.NoError(t, ctrl.authorizePrivilegedEntityWrites(memberCtx, &repo.EntityUpdate{PrivateToSelf: &yes}, false))

	uid := member.ID
	_, err := repos.Entities.UpdateByGroup(ctx, g.ID, repo.EntityUpdate{
		ID: item.ID, Name: item.Name, Description: item.Description, Quantity: item.Quantity,
		EntityTypeID: item.EntityType.ID, ParentID: item.Parent.ID, Notes: item.Notes,
		PrivateToSelf: &yes, PrivateToUserID: &uid,
	})
	require.NoError(t, err)

	_, err = repos.Entities.GetOneByGroup(ctx, g.ID, member.ID, item.ID)
	require.NoError(t, err)
	_, err = repos.Entities.GetOneByGroup(ctx, g.ID, owner.ID, item.ID)
	require.Error(t, err)

	list, err := repos.Entities.QueryByGroup(ctx, g.ID, owner.ID, repo.EntityQuery{Page: 1, PageSize: 30})
	require.NoError(t, err)
	for _, row := range list.Items {
		require.NotEqual(t, item.ID, row.ID)
	}
	listMine, err := repos.Entities.QueryByGroup(ctx, g.ID, member.ID, repo.EntityQuery{Page: 1, PageSize: 30})
	require.NoError(t, err)
	found := false
	for _, row := range listMine.Items {
		if row.ID == item.ID {
			found = true
		}
	}
	require.True(t, found)

	require.NoError(t, repos.Groups.SetMemberPrivileged(ctx, g.ID, member.ID, false))
	ordinary := withTenant(ctx, member, g.ID)
	err = ctrl.authorizePrivilegedEntityWrites(ordinary, &repo.EntityUpdate{PrivateToSelf: &yes}, false)
	require.Error(t, err)
}

func TestStaffNotesExcludedFromAISnapshot(t *testing.T) {
	ctx, ctrl, repos, g, owner, _, item := staffNotesFixture(t)
	secret := "do-not-leak-to-model"
	_, err := repos.Entities.UpdateByGroup(ctx, g.ID, repo.EntityUpdate{
		ID: item.ID, Name: item.Name, Description: item.Description, Quantity: item.Quantity,
		EntityTypeID: item.EntityType.ID, ParentID: item.Parent.ID, Notes: item.Notes, StaffNotes: &secret,
	})
	require.NoError(t, err)

	ownerCtx := withTenant(ctx, owner, g.ID)
	_, snapshot, err := ctrl.aiActionSource(ownerCtx, "item", item.ID)
	require.NoError(t, err)
	require.NotContains(t, snapshot, "staffNotes")
	require.NotContains(t, snapshot, secret)
}

func TestMembersPrivilegeLabelsHiddenFromOrdinary(t *testing.T) {
	ctx, ctrl, repos, g, owner, member, _ := staffNotesFixture(t)
	require.NoError(t, repos.Groups.SetMemberPrivileged(ctx, g.ID, member.ID, true))

	ownerAuth := services.SetTenantCtx(services.SetUserCtx(ctx, &owner, ""), g.ID)
	r := peekMembers(ctrl, ownerAuth)
	require.True(t, r.SelfPrivileged)
	require.True(t, r.SelfIsOwner)
	for _, m := range r.Members {
		require.NotNil(t, m.Privileged)
	}

	require.NoError(t, repos.Groups.SetMemberPrivileged(ctx, g.ID, member.ID, false))
	memberAuth := services.SetTenantCtx(services.SetUserCtx(ctx, &member, ""), g.ID)
	ordinary := peekMembers(ctrl, memberAuth)
	require.False(t, ordinary.SelfPrivileged)
	for _, m := range ordinary.Members {
		require.Nil(t, m.Privileged)
		require.Nil(t, m.IsOwner)
	}
}

func peekMembers(ctrl *V1Controller, ctx context.Context) repo.GroupMembersOut {
	auth := services.NewContext(ctx)
	selfPriv, _ := ctrl.repo.Groups.IsPrivilegedOf(auth, auth.UID, auth.GID)
	selfOwner, _ := ctrl.repo.Groups.IsOwnerOf(auth, auth.UID, auth.GID)
	members, _ := ctrl.repo.Users.GetMembersByGroupID(auth, auth.GID)
	if !selfPriv {
		for i := range members {
			members[i].Privileged = nil
			members[i].IsOwner = nil
		}
	}
	return repo.GroupMembersOut{Members: members, SelfPrivileged: selfPriv, SelfIsOwner: selfOwner}
}
