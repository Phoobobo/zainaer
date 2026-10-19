// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package v1

import (
	"errors"
	"net/http"

	"github.com/hay-kot/httpkit/errchain"
	"github.com/hay-kot/httpkit/server"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
)

func (ctrl *V1Controller) HandleCollectionNaming() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var body struct {
			Name string `json:"name"`
		}
		if err := server.Decode(r, &body); err != nil {
			return validate.NewRequestError(err, http.StatusBadRequest)
		}
		actor := services.UseUserCtx(r.Context())
		if err := ctrl.repo.Users.CompleteCollectionNaming(r.Context(), actor.ID, body.Name); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, repo.ErrCollectionNameInvalid) {
				status = http.StatusBadRequest
			}
			if errors.Is(err, repo.ErrCollectionNamingUnavailable) {
				status = http.StatusConflict
			}
			return validate.NewRequestError(err, status)
		}
		return server.JSON(w, http.StatusNoContent, nil)
	}
}
