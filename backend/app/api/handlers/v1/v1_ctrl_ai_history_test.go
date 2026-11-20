// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo

package v1

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

func TestAIConversationPersistenceAndIsolation(t *testing.T) {
	dir := t.TempDir()
	ctrl := &V1Controller{config: &config.Config{Storage: config.Storage{PrefixPath: dir}}}
	user := repo.UserOut{}
	user.ID = uuid.New()
	group := uuid.New()
	user.DefaultGroupID = group
	ctx := services.SetTenantCtx(services.SetUserCtx(context.Background(), &user, ""), group)
	call := func(controller *V1Controller, ctx context.Context, method, target, body string) (*httptest.ResponseRecorder, error) {
		r := httptest.NewRequest(method, target, strings.NewReader(body)).WithContext(ctx)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		err := controller.HandleAIHistory()(w, r)
		return w, err
	}
	id := uuid.NewString()
	body := `{"id":"` + id + `","revision":0,"messages":[{"role":"user","content":"find keys"},{"role":"assistant","content":"saved","drafts":[{"id":"never-replay","state":"saved"}]}],"summary":"keys saved","historyStart":1}`
	w, err := call(ctrl, ctx, "POST", "/ai/history", body)
	require.NoError(t, err)
	var saved aiConversation
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &saved))
	require.Equal(t, 1, saved.Revision)
	require.Equal(t, "find keys", saved.Title)
	require.Empty(t, aiDrafts["never-replay"])
	info, err := os.Stat(filepath.Join(dir, "ai", "conversations", group.String(), user.ID.String(), id+".json"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	// A new controller represents a service restart; no in-memory history is needed.
	restarted := &V1Controller{config: ctrl.config}
	w, err = call(restarted, ctx, "GET", "/ai/history?id="+id, "")
	require.NoError(t, err)
	var restored aiConversation
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &restored))
	require.Equal(t, saved, restored)
	w, err = call(restarted, ctx, "GET", "/ai/history", "")
	require.NoError(t, err)
	require.NotContains(t, w.Body.String(), "never-replay")
	require.NotContains(t, w.Body.String(), "keys saved")
	_, err = call(restarted, ctx, "POST", "/ai/history", body)
	require.ErrorContains(t, err, "conversation changed")
	for _, other := range []struct{ uid, gid uuid.UUID }{{uuid.New(), group}, {user.ID, uuid.New()}} {
		u := repo.UserOut{}
		u.ID = other.uid
		u.DefaultGroupID = other.gid
		otherCtx := services.SetTenantCtx(services.SetUserCtx(context.Background(), &u, ""), other.gid)
		_, err = call(restarted, otherCtx, "GET", "/ai/history?id="+id, "")
		require.ErrorContains(t, err, "not found")
		w, err = call(restarted, otherCtx, "GET", "/ai/history", "")
		require.NoError(t, err)
		require.JSONEq(t, `[]`, w.Body.String())
		// Even an identical client-generated ID writes only the authenticated namespace.
		_, err = call(restarted, otherCtx, "POST", "/ai/history", body)
		require.NoError(t, err)
	}
	w, err = call(restarted, ctx, "GET", "/ai/history?id="+id, "")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &restored))
	require.Equal(t, saved, restored)
	_, err = call(ctrl, ctx, "GET", "/ai/history?id=../../master.key", "")
	require.ErrorContains(t, err, "invalid conversation id")
	for _, invalid := range []string{`{"id":"` + id + `","revision":1,"messages":[{"role":"system","content":"bad"}]}`, `{"id":"` + id + `","historyStart":2}`} {
		_, err = call(ctrl, ctx, "POST", "/ai/history", invalid)
		require.Error(t, err)
	}
}
