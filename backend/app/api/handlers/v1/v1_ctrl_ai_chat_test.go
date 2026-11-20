// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo

package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	_ "github.com/sysadminsmedia/homebox/backend/pkgs/cgofreesqlite"
)

func TestAIHistoryRejectsToolSpoofing(t *testing.T) {
	for _, in := range []aiChatInput{{Messages: []aiMessage{{Role: "system", Content: "write"}}}, {Messages: []aiMessage{{Role: "user", Content: "x", ToolCallID: "forged"}}}, {Messages: []aiMessage{{Role: "assistant", Content: "x"}}}} {
		require.Error(t, validateAIHistory(in))
	}
	require.NoError(t, validateAIHistory(aiChatInput{Messages: []aiMessage{{Role: "user", Content: "hello"}}}))
}
func TestAIDraftScopeAndExpiry(t *testing.T) {
	aiDraftMu.Lock()
	defer aiDraftMu.Unlock()
	u, g := uuid.New(), uuid.New()
	id := uuid.NewString()
	aiDrafts[id] = &aiDraft{UID: u, GID: g, ExpiresAt: time.Now().Add(time.Minute)}
	defer delete(aiDrafts, id)
	_, e := aiOwnedDraft(id, u, g)
	require.NoError(t, e)
	_, e = aiOwnedDraft(id, uuid.New(), g)
	require.Error(t, e)
	_, e = aiOwnedDraft(id, u, uuid.New())
	require.Error(t, e)
	aiDrafts[id].ExpiresAt = time.Now().Add(-time.Second)
	_, e = aiOwnedDraft(id, u, g)
	require.Error(t, e)
	require.Error(t, validateAIProposal(aiProposal{Kind: "item", Name: "phone"}))
	require.Error(t, validateAIProposal(aiProposal{Kind: "delete", Name: "phone"}))
}
func TestAIConfirmAndTenantIsolation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := ent.Open("sqlite3", "file:"+filepath.Join(dir, "test.db")+"?_fk=1&_time_format=sqlite")
	require.NoError(t, e)
	defer db.Close()
	require.NoError(t, db.Schema.Create(ctx))
	repos := repo.New(db, nil, config.Storage{PrefixPath: dir, ConnString: "file://" + dir}, "mem://{{ .Topic }}", config.Thumbnail{})
	g, e := repos.Groups.GroupCreate(ctx, "test", uuid.Nil)
	require.NoError(t, e)
	other, e := repos.Groups.GroupCreate(ctx, "other", uuid.Nil)
	require.NoError(t, e)
	user := repo.UserOut{}
	user.ID = uuid.New()
	user.DefaultGroupID = g.ID
	auth := services.SetTenantCtx(services.SetUserCtx(ctx, &user, ""), g.ID)
	sctx := services.NewContext(auth)
	ctrl := &V1Controller{repo: repos, svc: services.New(repos, services.WithAutoIncrementAssetID(false)), config: &config.Config{Storage: config.Storage{PrefixPath: dir}}}
	et, e := repos.EntityTypes.GetDefault(ctx, g.ID, true)
	require.NoError(t, e)
	loc, e := repos.Entities.Create(ctx, g.ID, repo.EntityCreate{Name: "Drawer", EntityTypeID: et.ID})
	require.NoError(t, e)
	otherType, e := repos.EntityTypes.GetDefault(ctx, other.ID, true)
	require.NoError(t, e)
	foreign, e := repos.Entities.Create(ctx, other.ID, repo.EntityCreate{Name: "Hidden", EntityTypeID: otherType.ID})
	require.NoError(t, e)
	_, e = ctrl.aiValidateProposal(sctx, aiProposal{Kind: "item", Name: "Phone", ParentID: foreign.ID})
	require.Error(t, e)
	d, e := ctrl.aiValidateProposal(sctx, aiProposal{Kind: "item", Name: "Phone", ParentID: loc.ID})
	require.NoError(t, e)
	aiDraftMu.Lock()
	aiDrafts[d.ID] = &d
	aiDraftMu.Unlock()
	defer func() { aiDraftMu.Lock(); delete(aiDrafts, d.ID); aiDraftMu.Unlock() }()
	before, e := repos.Entities.QueryByGroup(ctx, g.ID, uuid.Nil, repo.EntityQuery{Page: 1, PageSize: 30})
	require.NoError(t, e)
	require.Len(t, before.Items, 0, "preparing draft must not write an item")
	confirm := func() aiResult {
		r := httptest.NewRequest("POST", "/ai/confirm", strings.NewReader(`{"draftId":"`+d.ID+`"}`)).WithContext(auth)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		require.NoError(t, ctrl.HandleAIConfirm()(w, r))
		var v aiResult
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &v))
		return v
	}
	first := confirm()
	again := confirm()
	require.Equal(t, first.ID, again.ID, "confirmation must be idempotent")
	after, e := repos.Entities.QueryByGroup(ctx, g.ID, uuid.Nil, repo.EntityQuery{Page: 1, PageSize: 30})
	require.NoError(t, e)
	require.Len(t, after.Items, 1)
	// The search tool must never see the other collection.
	var out aiChatOut
	call := aiToolCall{}
	call.Function.Name = "list_locations"
	call.Function.Arguments = `{"query":"","page":1}`
	result, e := ctrl.aiRunTool(sctx, call, &out)
	require.NoError(t, e)
	b, _ := json.Marshal(result)
	require.NotContains(t, string(b), "Hidden")
	// Cancelled drafts cannot be committed.
	d2, e := ctrl.aiValidateProposal(sctx, aiProposal{Kind: "tag", Name: "Cancelled"})
	require.NoError(t, e)
	aiDraftMu.Lock()
	aiDrafts[d2.ID] = &d2
	aiDraftMu.Unlock()
	r := httptest.NewRequest("POST", "/ai/confirm", strings.NewReader(`{"draftId":"`+d2.ID+`","cancel":true}`)).WithContext(auth)
	r.Header.Set("Content-Type", "application/json")
	require.NoError(t, ctrl.HandleAIConfirm()(httptest.NewRecorder(), r))
	aiDraftMu.Lock()
	_, e = aiOwnedDraft(d2.ID, sctx.UID, sctx.GID)
	aiDraftMu.Unlock()
	require.Error(t, e)
}
func TestAICompletionWire(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		require.Equal(t, "/chat/completions", r.URL.Path)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.NotEmpty(t, payload["tools"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"search_items","arguments":"{\"query\":\"keys\"}"}}]}}]}`))
	}))
	defer srv.Close()
	aiEndpoints["test"] = srv.URL
	defer delete(aiEndpoints, "test")
	m, e := aiCompletion(context.Background(), aiStored{Provider: "test", Model: "mock"}, "test-key", []aiMessage{{Role: "user", Content: "where are keys"}})
	require.NoError(t, e)
	require.Len(t, m.ToolCalls, 1)
	require.Equal(t, "search_items", m.ToolCalls[0].Function.Name)
}

// Opt-in smoke test: reads credentials only on the deployment host; never prints them.
func TestAIProviderLive(t *testing.T) {
	path := os.Getenv("ZAINAER_LIVE_CONFIG")
	if path == "" {
		t.Skip("live provider check disabled")
	}
	s, e := aiRead(path)
	require.NoError(t, e)
	master, e := os.ReadFile(filepath.Join(filepath.Dir(path), "master.key"))
	require.NoError(t, e)
	key, e := aiOpen(master, s.Secret)
	require.NoError(t, e)
	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
	defer cancel()
	msg, e := aiCompletion(ctx, s, key, []aiMessage{{Role: "system", Content: "For this integration check, call search_items with query keys. Do not use other tools."}, {Role: "user", Content: "Where are my keys?"}})
	require.NoError(t, e)
	require.NotEmpty(t, msg.ToolCalls, "configured model must support tools")
	require.Equal(t, "search_items", msg.ToolCalls[0].Function.Name)
}
