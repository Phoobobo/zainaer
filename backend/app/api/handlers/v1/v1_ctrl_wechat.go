// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package v1

import (
	"errors"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/hay-kot/httpkit/server"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/wechat"
	"net/http"
)

// HandleWeChatLogin accepts only a short-lived code. Client-supplied OpenID,
// user ID, email and collection IDs never establish identity.
func (ctrl *V1Controller) HandleWeChatLogin(bind bool) errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		conf := ctrl.config.WeChat
		if !conf.Enabled || conf.AppID == "" || conf.AppSecret == "" || ctrl.isDemo {
			return validate.NewRequestError(errors.New("wechat login is not configured"), http.StatusServiceUnavailable)
		}
		if bind && services.IsAPIKeyAuth(r.Context()) {
			return validate.NewRequestError(errors.New("session login required to link wechat"), http.StatusForbidden)
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var in struct {
			Code     string `json:"code"`
			Register bool   `json:"register"`
		}
		if err := server.Decode(r, &in); err != nil || in.Code == "" || len(in.Code) > 512 {
			return validate.NewRequestError(errors.New("invalid login code"), http.StatusBadRequest)
		}
		openID, err := wechat.Exchange(r.Context(), nil, conf.AppID, conf.AppSecret, in.Code)
		if err != nil {
			return validate.NewUnauthorizedError()
		}
		identity := services.WeChatIdentity(conf.AppID, openID)
		if bind {
			user := services.UseUserCtx(r.Context())
			if user == nil {
				return validate.NewUnauthorizedError()
			}
			if err := ctrl.svc.User.BindWeChat(r.Context(), user.ID, identity); err != nil {
				return validate.NewRequestError(errors.New("account or wechat identity is already linked"), http.StatusConflict)
			}
			return server.JSON(w, http.StatusNoContent, nil)
		}
		token, err := ctrl.svc.User.LoginWeChat(r.Context(), identity, in.Register && ctrl.allowRegistration)
		if errors.Is(err, services.ErrWeChatUnbound) {
			return server.JSON(w, http.StatusConflict, map[string]interface{}{"code": "wechat_unbound", "registrationAllowed": ctrl.allowRegistration})
		}
		if err != nil {
			return validate.NewUnauthorizedError()
		}
		w.Header().Set("Cache-Control", "no-store")
		return server.JSON(w, http.StatusOK, TokenResponse{Token: "Bearer " + token.Raw, ExpiresAt: token.ExpiresAt, AttachmentToken: token.AttachmentToken})
	}
}
