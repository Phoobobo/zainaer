// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/migrations"
	_ "github.com/sysadminsmedia/homebox/backend/pkgs/cgofreesqlite"
)

func TestCollectionNamingUpgradePreservesExistingUsers(t *testing.T) {
	db, err := ent.Open("sqlite3", "file:"+filepath.Join(t.TempDir(), "upgrade.db")+"?_fk=1&_time_format=sqlite")
	require.NoError(t, err)
	defer db.Close()
	files, err := migrations.Migrations("sqlite3")
	require.NoError(t, err)
	goose.SetBaseFS(files)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(db.Sql(), "sqlite3", 20260512130000))
	uid := uuid.New()
	_, err = db.Sql().Exec(`INSERT INTO users (id, name, email, created_at, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, uid.String(), "Existing", "existing@example.test")
	require.NoError(t, err)
	require.NoError(t, runMigrations(db, "sqlite3"))
	usr, err := db.User.Get(context.Background(), uid)
	require.NoError(t, err)
	require.Equal(t, "Existing", usr.Name)
	require.Nil(t, usr.CollectionNamingID)
	key, err := db.Passkey.Create().SetUserID(uid).SetName("migration test").SetCredentialID("test-id").SetCredential([]byte("{}")).Save(context.Background())
	require.NoError(t, err)
	require.Equal(t, uid, key.UserID)
	require.NoError(t, db.User.DeleteOneID(uid).Exec(context.Background()))
	n, err := db.Passkey.Query().Count(context.Background())
	require.NoError(t, err)
	require.Zero(t, n, "deleting a user cascades to credentials")
	require.NoError(t, runMigrations(db, "sqlite3"))
}
