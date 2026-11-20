// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo

package v1

import (
	"errors"
	"net/http"

	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
)

// filterEntityStaffNotes strips privileged-only fields for ordinary members and
// keeps controls visible for privileged members.
func (ctrl *V1Controller) filterEntityStaffNotes(ctx services.Context, out repo.EntityOut) (repo.EntityOut, error) {
	ok, err := ctrl.repo.Groups.IsPrivilegedOf(ctx, ctx.UID, ctx.GID)
	if err != nil {
		return out, validate.NewRequestError(errors.New("cannot verify membership privilege"), http.StatusInternalServerError)
	}
	repo.ApplyPrivilegedFieldsVisibility(&out, ok)
	return out, nil
}

// authorizePrivilegedEntityWrites rejects non-privileged attempts to set staff
// notes or private-to-self. Nil pointers mean "leave unchanged".
func (ctrl *V1Controller) authorizePrivilegedEntityWrites(ctx services.Context, body *repo.EntityUpdate, isLocation bool) error {
	if body == nil {
		return nil
	}
	needsPrivilege := body.StaffNotes != nil || body.PrivateToSelf != nil
	if !needsPrivilege {
		return nil
	}
	ok, err := ctrl.repo.Groups.IsPrivilegedOf(ctx, ctx.UID, ctx.GID)
	if err != nil {
		return validate.NewRequestError(errors.New("cannot verify membership privilege"), http.StatusInternalServerError)
	}
	if !ok {
		return validate.NewRequestError(errors.New("only privileged members can edit staff notes or private visibility"), http.StatusForbidden)
	}
	if body.PrivateToSelf != nil {
		if isLocation {
			return validate.NewRequestError(errors.New("private visibility applies only to items"), http.StatusBadRequest)
		}
		if *body.PrivateToSelf {
			uid := ctx.UID
			body.PrivateToUserID = &uid
		}
	}
	return nil
}

// authorizeStaffNotesWrite is retained for focused staff-notes tests.
func (ctrl *V1Controller) authorizeStaffNotesWrite(ctx services.Context, body *repo.EntityUpdate) error {
	return ctrl.authorizePrivilegedEntityWrites(ctx, body, false)
}
