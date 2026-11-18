// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package repo

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/pkgs/hasher"
)

func TestInvitationLifecycle(t *testing.T) {
	ctx := context.Background()
	for _, state := range []string{"valid", "deleted", "missing", "expired", "exhausted", "member"} {
		t.Run(state, func(t *testing.T) {
			g, err := tRepos.Groups.GroupCreate(ctx, "invitation-"+state, uuid.Nil)
			require.NoError(t, err)
			token := hasher.GenerateToken()
			expiry, uses := time.Now().Add(time.Hour), 1
			if state == "expired" {
				expiry = time.Now().Add(-time.Hour)
			}
			if state == "exhausted" {
				uses = 0
			}
			inv, err := tRepos.Groups.InvitationCreate(ctx, g.ID, GroupInvitationCreate{Token: token.Hash, ExpiresAt: expiry, Uses: uses})
			require.NoError(t, err)
			if state == "deleted" {
				require.NoError(t, tRepos.Groups.InvitationDelete(ctx, g.ID, inv.ID))
			}
			if state == "missing" {
				token = hasher.GenerateToken()
			}
			if state == "member" {
				require.NoError(t, tClient.Group.UpdateOneID(g.ID).AddUserIDs(tUser.ID).Exec(ctx))
			}
			joined, err := tRepos.Groups.InvitationAccept(ctx, token.Hash, tUser.ID)
			expected := map[string]error{"deleted": ErrInvitationNotFound, "missing": ErrInvitationNotFound, "expired": ErrInvitationExpired, "exhausted": ErrInvitationUsedUp, "member": ErrAlreadyGroupMember}
			if state == "valid" {
				require.NoError(t, err)
				require.Equal(t, g.ID, joined.ID)
				found, err := tRepos.Groups.InvitationGet(ctx, token.Hash)
				require.NoError(t, err)
				require.Zero(t, found.Uses)
			} else {
				require.ErrorIs(t, err, expected[state])
				member, err := tRepos.Groups.IsMember(ctx, g.ID, tUser.ID)
				require.NoError(t, err)
				require.Equal(t, state == "member", member)
			}
		})
	}
}
