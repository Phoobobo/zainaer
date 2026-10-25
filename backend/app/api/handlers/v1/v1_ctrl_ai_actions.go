// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo

package v1

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

// Only server-resolved records become editable proposals. The snapshot prevents
// a stale card from overwriting edits made after it was prepared.
func (ctrl *V1Controller) aiActionSource(ctx services.Context, kind string, id uuid.UUID) (aiResult, string, error) {
	if id == uuid.Nil {
		return aiResult{}, "", errors.New("existing target ID required")
	}
	if kind == "tag" {
		v, err := ctrl.repo.Tags.GetOneByGroup(ctx, ctx.GID, id)
		if err != nil {
			return aiResult{}, "", errors.New("tag unavailable in this collection")
		}
		b, _ := json.Marshal(v.TagSummary)
		return aiResult{ID: v.ID, Kind: "tag", Name: v.Name, Description: v.Description, Tags: []string{}}, string(b), nil
	}
	if kind != "item" && kind != "location" {
		return aiResult{}, "", errors.New("unsupported edit kind")
	}
	v, err := ctrl.repo.Entities.GetOneByGroup(ctx, ctx.GID, id)
	if err != nil || v.EntityType == nil || v.EntityType.IsLocation != (kind == "location") {
		return aiResult{}, "", errors.New("target unavailable or wrong kind")
	}
	b, _ := json.Marshal(v)
	return ctrl.aiEntityResult(ctx, v.EntitySummary), string(b), nil
}

func (ctrl *V1Controller) aiPrepareActions(ctx services.Context, args string, out *aiChatOut) (any, error) {
	var in struct {
		Actions []struct {
			Kind        string       `json:"kind"`
			TargetID    uuid.UUID    `json:"targetId"`
			Name        *string      `json:"name"`
			Description *string      `json:"description"`
			ParentID    *string      `json:"parentId"`
			TagIDs      *[]uuid.UUID `json:"tagIds"`
		} `json:"actions"`
	}
	decoder := json.NewDecoder(strings.NewReader(args))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil || len(in.Actions) == 0 || len(in.Actions) > 100 {
		return nil, errors.New("provide 1 to 100 valid actions")
	}
	drafts := []aiDraft{}
	seen := map[uuid.UUID]bool{}
	for _, a := range in.Actions {
		if seen[a.TargetID] {
			return nil, errors.New("duplicate action target")
		}
		seen[a.TargetID] = true
		_, _, err := ctrl.aiActionSource(ctx, a.Kind, a.TargetID)
		if err != nil {
			return nil, err
		}
		p := aiProposal{Action: "update", Kind: a.Kind, TargetID: a.TargetID}
		if a.Kind == "tag" {
			old, err := ctrl.repo.Tags.GetOneByGroup(ctx, ctx.GID, a.TargetID)
			if err != nil {
				return nil, err
			}
			p.Name, p.Description, p.ParentID = old.Name, old.Description, old.ParentID
		} else {
			old, err := ctrl.repo.Entities.GetOneByGroup(ctx, ctx.GID, a.TargetID)
			if err != nil {
				return nil, err
			}
			p.Name, p.Description = old.Name, old.Description
			if old.Parent != nil {
				p.ParentID = old.Parent.ID
			}
			p.TagIDs = []uuid.UUID{}
			for _, t := range old.Tags {
				p.TagIDs = append(p.TagIDs, t.ID)
			}
		}
		if a.Name == nil && a.Description == nil && a.ParentID == nil && a.TagIDs == nil {
			return nil, errors.New("action has no edits")
		}
		if a.Name != nil {
			p.Name = strings.TrimSpace(*a.Name)
		}
		if a.Description != nil {
			p.Description = *a.Description
		}
		if a.ParentID != nil {
			p.ParentID = uuid.Nil
			if *a.ParentID != "" {
				p.ParentID, err = uuid.Parse(*a.ParentID)
				if err != nil {
					return nil, errors.New("invalid destination")
				}
			}
		}
		if a.TagIDs != nil {
			if a.Kind == "tag" {
				return nil, errors.New("tags cannot have tags")
			}
			p.TagIDs = *a.TagIDs
		}
		d, err := ctrl.aiValidateProposal(ctx, p)
		if err != nil {
			return nil, err
		}
		drafts = append(drafts, d)
	}
	aiDraftMu.Lock()
	defer aiDraftMu.Unlock()
	if len(aiDrafts)+len(drafts) > 1000 {
		return nil, errors.New("draft capacity reached")
	}
	for i := range drafts {
		d := drafts[i]
		aiDrafts[d.ID] = &d
	}
	out.Drafts = append(out.Drafts, drafts...)
	// Matched records are represented by their review cards during an edit.
	out.Items = []aiResult{}
	return map[string]any{"status": "awaiting_user_confirmation", "actions": drafts}, nil
}

func (ctrl *V1Controller) aiApplyEdit(ctx services.Context, p aiProposal) (aiResult, error) {
	if p.Kind == "tag" {
		old, err := ctrl.repo.Tags.GetOneByGroup(ctx, ctx.GID, p.TargetID)
		if err != nil {
			return aiResult{}, err
		}
		v, err := ctrl.repo.Tags.UpdateByGroup(ctx, ctx.GID, repo.TagUpdate{ID: p.TargetID, Name: p.Name, Description: p.Description, ParentID: p.ParentID, Color: old.Color, Icon: old.Icon})
		return aiResult{ID: v.ID, Kind: "tag", Name: v.Name, Description: v.Description}, err
	}
	old, err := ctrl.repo.Entities.GetOneByGroup(ctx, ctx.GID, p.TargetID)
	if err != nil {
		return aiResult{}, err
	}
	// Copy the complete record before changing editable household fields so photos,
	// custom fields, quantity and other existing metadata survive a move.
	update := repo.EntityUpdate{SyncChildEntityLocations: old.SyncChildEntityLocations, AssetID: old.AssetID, Quantity: old.Quantity, Insured: old.Insured, Archived: old.Archived, SerialNumber: old.SerialNumber, ModelNumber: old.ModelNumber, Manufacturer: old.Manufacturer, LifetimeWarranty: old.LifetimeWarranty, WarrantyExpires: old.WarrantyExpires, WarrantyDetails: old.WarrantyDetails, PurchaseDate: old.PurchaseDate, PurchaseFrom: old.PurchaseFrom, PurchasePrice: old.PurchasePrice, SoldDate: old.SoldDate, SoldTo: old.SoldTo, SoldPrice: old.SoldPrice, SoldNotes: old.SoldNotes, Notes: old.Notes, Fields: old.Fields}
	update.ID, update.Name, update.Description, update.ParentID, update.TagIDs = p.TargetID, p.Name, p.Description, p.ParentID, p.TagIDs
	update.EntityTypeID = old.EntityType.ID
	v, err := ctrl.repo.Entities.UpdateByGroup(ctx, ctx.GID, update)
	if err != nil {
		return aiResult{}, err
	}
	return ctrl.aiEntityResult(ctx, v.EntitySummary), nil
}
