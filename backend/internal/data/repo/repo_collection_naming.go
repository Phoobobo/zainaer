// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/usergroup"
)

var ErrCollectionNamingUnavailable = errors.New("collection naming is not pending for an owned collection")
var ErrCollectionNameInvalid = errors.New("collection name must contain 1 to 255 bytes")

// CompleteCollectionNaming binds the change to the authenticated user's initial
// collection, not a client-supplied tenant. The name and completion are atomic.
func (r *UserRepository) CompleteCollectionNaming(ctx context.Context, uid uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 255 {
		return ErrCollectionNameInvalid
	}
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	usr, err := tx.User.Get(ctx, uid)
	if err != nil {
		return err
	}
	if usr.CollectionNamingID == nil {
		return ErrCollectionNamingUnavailable
	}
	gid := *usr.CollectionNamingID
	owner, err := tx.UserGroup.Query().Where(usergroup.UserID(uid), usergroup.GroupID(gid), usergroup.RoleEQ(usergroup.RoleOwner)).Exist(ctx)
	if err != nil {
		return err
	}
	if !owner {
		return ErrCollectionNamingUnavailable
	}
	if err := tx.Group.UpdateOneID(gid).SetName(name).Exec(ctx); err != nil {
		return err
	}
	if err := tx.User.UpdateOneID(uid).ClearCollectionNamingID().Exec(ctx); err != nil {
		return err
	}
	return tx.Commit()
}
