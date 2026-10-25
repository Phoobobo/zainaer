// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package repo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/passkey"
	"time"
)

func (r *UserRepository) Passkeys(ctx context.Context, uid uuid.UUID) ([]*ent.Passkey, error) {
	return r.db.Passkey.Query().Where(passkey.UserID(uid)).All(ctx)
}
func (r *UserRepository) AddPasskey(ctx context.Context, uid uuid.UUID, name string, c *webauthn.Credential) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = r.db.Passkey.Create().SetUserID(uid).SetName(name).SetCredentialID(base64.RawURLEncoding.EncodeToString(c.ID)).SetCredential(data).Save(ctx)
	return err
}
func (r *UserRepository) FindPasskey(ctx context.Context, id []byte, uid uuid.UUID) (*ent.Passkey, error) {
	return r.db.Passkey.Query().Where(passkey.CredentialID(base64.RawURLEncoding.EncodeToString(id)), passkey.UserID(uid)).Only(ctx)
}

// Compare-and-swap prevents replay races and resurrecting deleted credentials.
func (r *UserRepository) UsePasskey(ctx context.Context, old *ent.Passkey, c *webauthn.Credential) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	n, err := r.db.Passkey.Update().Where(passkey.ID(old.ID), passkey.UserID(old.UserID), passkey.Version(old.Version)).SetCredential(data).SetLastUsedAt(time.Now()).AddVersion(1).Save(ctx)
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("credential changed or removed")
	}
	return nil
}
func (r *UserRepository) DeletePasskey(ctx context.Context, uid, id uuid.UUID) error {
	n, err := r.db.Passkey.Delete().Where(passkey.ID(id), passkey.UserID(uid)).Exec(ctx)
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("credential not found")
	}
	return nil
}
