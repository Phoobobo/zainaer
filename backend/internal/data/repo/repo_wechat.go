// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package repo

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/user"
)

func (r *UserRepository) GetWeChat(ctx context.Context, identity string) (UserOut, error) {
	u, err := r.db.User.Query().Where(user.WechatIdentity(identity)).Only(ctx)
	if err != nil {
		return UserOut{}, err
	}
	return r.GetOneID(ctx, u.ID)
}

func (r *UserRepository) CreateWeChat(ctx context.Context, in UserCreate, identity string) (UserOut, error) {
	id, err := r.createUserWithMembership(ctx, in, func(q *ent.UserCreate) *ent.UserCreate { return q.SetWechatIdentity(identity) })
	if err != nil {
		return UserOut{}, err
	}
	return r.GetOneID(ctx, id)
}

// BindWeChat is a compare-and-set: neither a user nor a WeChat identity can
// silently be rebound. The unique index arbitrates concurrent requests.
func (r *UserRepository) BindWeChat(ctx context.Context, id uuid.UUID, identity string) error {
	n, err := r.db.User.Update().Where(user.ID(id), user.Or(user.WechatIdentityIsNil(), user.WechatIdentity(identity))).SetWechatIdentity(identity).Save(ctx)
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("wechat identity already bound")
	}
	return nil
}
