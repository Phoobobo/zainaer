// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo
package v1

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

func TestAIBatchEditActions(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := ent.Open("sqlite3", "file:"+filepath.Join(dir, "actions.db")+"?_fk=1&_time_format=sqlite")
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.Schema.Create(ctx))
	repos := repo.New(db, nil, config.Storage{PrefixPath: dir, ConnString: "file://" + dir}, "mem://{{ .Topic }}", config.Thumbnail{})
	g, err := repos.Groups.GroupCreate(ctx, "actions", uuid.Nil)
	require.NoError(t, err)
	user := repo.UserOut{}
	user.ID = uuid.New()
	user.DefaultGroupID = g.ID
	auth := services.SetTenantCtx(services.SetUserCtx(ctx, &user, ""), g.ID)
	sctx := services.NewContext(auth)
	ctrl := &V1Controller{repo: repos, svc: services.New(repos, services.WithAutoIncrementAssetID(false))}
	lt, err := repos.EntityTypes.GetDefault(ctx, g.ID, true)
	require.NoError(t, err)
	it, err := repos.EntityTypes.GetDefault(ctx, g.ID, false)
	require.NoError(t, err)
	source, err := repos.Entities.Create(ctx, g.ID, repo.EntityCreate{Name: "Old drawer", EntityTypeID: lt.ID})
	require.NoError(t, err)
	dest, err := repos.Entities.Create(ctx, g.ID, repo.EntityCreate{Name: "TV right drawer", EntityTypeID: lt.ID})
	require.NoError(t, err)
	tag, err := repos.Tags.Create(ctx, g.ID, repo.TagCreate{Name: "Camera"})
	require.NoError(t, err)
	ids := []uuid.UUID{}
	actions := []map[string]any{}
	for _, name := range []string{"Two cameras", "Action camera", "Camera mount"} {
		item, err := repos.Entities.Create(ctx, g.ID, repo.EntityCreate{Name: name, Description: "Preserve description", Quantity: 2, EntityTypeID: it.ID, ParentID: source.ID, TagIDs: []uuid.UUID{tag.ID}})
		require.NoError(t, err)
		_, err = repos.Entities.UpdateByGroup(ctx, g.ID, repo.EntityUpdate{ID: item.ID, Name: name, Description: item.Description, Quantity: 2, EntityTypeID: it.ID, ParentID: source.ID, TagIDs: []uuid.UUID{tag.ID}, SerialNumber: "serial-123", Notes: "Keep notes", Fields: []repo.EntityFieldData{{Type: "text", Name: "Lens", TextValue: "Wide"}}})
		require.NoError(t, err)
		ids = append(ids, item.ID)
		actions = append(actions, map[string]any{"kind": "item", "targetId": item.ID, "parentId": dest.ID.String()})
	}
	b, err := json.Marshal(map[string]any{"actions": actions})
	require.NoError(t, err)
	out := aiChatOut{}
	_, err = ctrl.aiPrepareActions(sctx, string(b), &out)
	require.NoError(t, err)
	require.Len(t, out.Drafts, 3)
	defer func() {
		aiDraftMu.Lock()
		defer aiDraftMu.Unlock()
		for _, d := range out.Drafts {
			delete(aiDrafts, d.ID)
		}
	}()
	for _, id := range ids {
		v, err := repos.Entities.GetOneByGroup(ctx, g.ID, id)
		require.NoError(t, err)
		require.Equal(t, source.ID, v.Parent.ID, "preparing must never write")
	}
	confirm := func(d aiDraft, cancel bool) (aiResult, error) {
		body, _ := json.Marshal(map[string]any{"draftId": d.ID, "cancel": cancel})
		r := httptest.NewRequest("POST", "/ai/confirm", strings.NewReader(string(body))).WithContext(auth)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		err := ctrl.HandleAIConfirm()(w, r)
		var result aiResult
		if err == nil {
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
		}
		return result, err
	}
	for _, d := range out.Drafts[:2] {
		require.NotNil(t, d.Before)
		require.Contains(t, d.Before.Path, "Old drawer")
		require.Equal(t, "TV right drawer", d.ParentPath)
		result, err := confirm(d, false)
		require.NoError(t, err)
		require.Equal(t, d.Proposal.TargetID, result.ID)
		again, err := confirm(d, false)
		require.NoError(t, err)
		require.Equal(t, result, again)
		v, err := repos.Entities.GetOneByGroup(ctx, g.ID, result.ID)
		require.NoError(t, err)
		require.Equal(t, dest.ID, v.Parent.ID)
		require.Equal(t, 2.0, v.Quantity)
		require.Equal(t, "serial-123", v.SerialNumber)
		require.Equal(t, "Keep notes", v.Notes)
		require.Len(t, v.Fields, 1)
		require.Equal(t, "Wide", v.Fields[0].TextValue)
		require.Equal(t, "Preserve description", v.Description)
		require.Len(t, v.Tags, 1)
		require.Equal(t, tag.ID, v.Tags[0].ID)
	}
	_, err = confirm(out.Drafts[2], true)
	require.NoError(t, err)
	_, err = confirm(out.Drafts[2], false)
	require.Error(t, err)
	v, err := repos.Entities.GetOneByGroup(ctx, g.ID, ids[2])
	require.NoError(t, err)
	require.Equal(t, source.ID, v.Parent.ID)
	bad := aiChatOut{}
	_, err = ctrl.aiPrepareActions(sctx, strings.Replace(string(b), ids[2].String(), uuid.NewString(), 1), &bad)
	require.Error(t, err)
	require.Empty(t, bad.Drafts)
	foreign, err := repos.Groups.GroupCreate(ctx, "foreign", uuid.Nil)
	require.NoError(t, err)
	foreignCtx := sctx
	foreignCtx.GID = foreign.ID
	_, _, err = ctrl.aiActionSource(foreignCtx, "item", ids[0])
	require.Error(t, err)
	stale := aiChatOut{}
	_, err = ctrl.aiPrepareActions(sctx, string(b), &stale)
	require.NoError(t, err)
	defer func() {
		aiDraftMu.Lock()
		defer aiDraftMu.Unlock()
		for _, d := range stale.Drafts {
			delete(aiDrafts, d.ID)
		}
	}()
	_, err = ctrl.aiApplyEdit(sctx, aiProposal{Action: "update", Kind: "item", TargetID: ids[0], Name: "Renamed camera", Description: "Other editor", ParentID: dest.ID, TagIDs: []uuid.UUID{tag.ID}})
	require.NoError(t, err)
	_, err = confirm(stale.Drafts[0], false)
	require.Error(t, err)
	_, err = ctrl.aiValidateProposal(sctx, aiProposal{Action: "update", Kind: "location", TargetID: dest.ID, Name: dest.Name, ParentID: dest.ID})
	require.Error(t, err)
}
