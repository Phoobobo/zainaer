// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hay-kot/httpkit/errchain"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
)

const (
	// phooboboAccountIssuer is the permanent issuer for shared phoobobo accounts.
	phooboboAccountIssuer = "https://account.phoobobo.top"
	phooboboAccountHost   = "account.phoobobo.top"
	phooboboCookie        = "phoobobo_sid"
	phooboboLegacyCookie  = "zjsy_sid"
)

var errPhooboboSignedOut = errors.New("phoobobo account is not signed in")

// phooboboAccountClient does not follow redirects. A redirect would forward
// the session cookie to whatever host the account service named.
var phooboboAccountClient = &http.Client{
	Timeout: 8 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type phooboboAccount struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Guest       bool   `json:"guest"`
	Anonymous   bool   `json:"anonymous"`
	Issuer      string `json:"issuer"`
	Subject     string `json:"subject"`
}

// HandlePhooboboLogin exchanges a shared phoobobo session cookie for a local
// session. A missing, guest, or anonymous session redirects to the account
// login page. Other failures return the user to the normal login page.
func (ctrl *V1Controller) HandlePhooboboLogin() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if !ctrl.config.Phoobobo.Enabled() {
			return validate.NewRequestError(fmt.Errorf("phoobobo account login is not enabled"), http.StatusForbidden)
		}

		acct, err := ctrl.fetchPhooboboAccount(r)
		if err != nil {
			if errors.Is(err, errPhooboboSignedOut) {
				log.Debug().Msg("phoobobo session missing; sending user to account login")
				http.Redirect(w, r, ctrl.phooboboLoginURL(r), http.StatusFound)
				return nil
			}
			log.Warn().Err(err).Msg("phoobobo account session check failed; returning to local login")
			http.Redirect(w, r, "/", http.StatusFound)
			return nil
		}

		email := strings.ToLower(strings.TrimSpace(acct.Email))
		if email == "" {
			if phone := strings.TrimSpace(acct.Phone); phone != "" {
				email = strings.ToLower(phone) + "@users.phoobobo.top"
			}
		}
		if email == "" {
			log.Warn().Msg("phoobobo account has no email or phone; returning to local login")
			http.Redirect(w, r, "/", http.StatusFound)
			return nil
		}
		subject := strings.TrimSpace(acct.Subject)
		if subject == "" {
			subject = fmt.Sprintf("phoobobo:%d", acct.ID)
		}
		name := strings.TrimSpace(acct.DisplayName)
		if name == "" {
			name = email
		}

		token, err := ctrl.svc.User.LoginOIDC(r.Context(), phooboboAccountIssuer, subject, email, name)
		if err != nil {
			log.Err(err).Msg("phoobobo account login failed; returning to local login")
			http.Redirect(w, r, "/", http.StatusFound)
			return nil
		}
		ctrl.setCookies(w, noPort(r.Host), token.Raw, token.ExpiresAt, true, token.AttachmentToken)
		http.Redirect(w, r, "/home", http.StatusFound)
		return nil
	}
}

// HandlePhooboboLogout clears the local session and sends the browser to the
// account logout page, which clears the shared cookie and returns to Zainaer.
func (ctrl *V1Controller) HandlePhooboboLogout() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if ctrl.svc != nil {
			if c, err := r.Cookie(cookieNameToken); err == nil && strings.TrimSpace(c.Value) != "" {
				if err := ctrl.svc.User.Logout(r.Context(), c.Value); err != nil {
					log.Warn().Err(err).Msg("local logout before account logout failed")
				}
			}
		}
		ctrl.unsetCookies(w, noPort(r.Host))
		if !ctrl.config.Phoobobo.Enabled() {
			http.Redirect(w, r, "/", http.StatusFound)
			return nil
		}
		http.Redirect(w, r, ctrl.phooboboLogoutURL(r), http.StatusFound)
		return nil
	}
}

func (ctrl *V1Controller) fetchPhooboboAccount(r *http.Request) (*phooboboAccount, error) {
	cookies := phooboboSessionCookies(r)
	if len(cookies) == 0 {
		return nil, errPhooboboSignedOut
	}
	var last error
	for _, c := range cookies {
		acct, err := ctrl.fetchPhooboboAccountWithCookie(r, c)
		if err == nil {
			return acct, nil
		}
		if errors.Is(err, errPhooboboSignedOut) {
			last = err
			continue
		}
		return nil, err
	}
	if last == nil {
		last = errPhooboboSignedOut
	}
	return nil, last
}

func (ctrl *V1Controller) fetchPhooboboAccountWithCookie(r *http.Request, c *http.Cookie) (*phooboboAccount, error) {
	endpoint, overrideHost, err := ctrl.phooboboSessionEndpoint()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if overrideHost != "" {
		req.Host = overrideHost
	}
	req.Header.Set("Accept", "application/json")
	req.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value})

	resp, err := phooboboAccountClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errPhooboboSignedOut
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("account session status %d", resp.StatusCode)
	}
	var acct phooboboAccount
	if err := json.Unmarshal(body, &acct); err != nil {
		return nil, err
	}
	if acct.Guest || acct.Anonymous || (acct.ID == 0 && strings.TrimSpace(acct.Subject) == "") {
		return nil, errPhooboboSignedOut
	}
	if !phooboboIssuerOK(acct.Issuer) {
		return nil, fmt.Errorf("unexpected phoobobo issuer")
	}
	return &acct, nil
}

func phooboboSessionCookies(r *http.Request) []*http.Cookie {
	var out []*http.Cookie
	for _, name := range []string{phooboboCookie, phooboboLegacyCookie} {
		c, err := r.Cookie(name)
		if err != nil || strings.TrimSpace(c.Value) == "" {
			continue
		}
		out = append(out, &http.Cookie{Name: name, Value: c.Value})
	}
	return out
}

func phooboboIssuerOK(raw string) bool {
	return strings.TrimRight(strings.TrimSpace(raw), "/") == phooboboAccountIssuer
}

func (ctrl *V1Controller) phooboboSessionEndpoint() (string, string, error) {
	public := strings.TrimRight(strings.TrimSpace(ctrl.config.Phoobobo.AccountURL), "/")
	internal := strings.TrimRight(strings.TrimSpace(ctrl.config.Phoobobo.AccountInternal), "/")
	base := public
	overrideHost := ""
	if internal != "" {
		base = internal
		overrideHost = phooboboUpstreamHost(public)
	}
	if base == "" {
		return "", "", fmt.Errorf("phoobobo account url is empty")
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", "", fmt.Errorf("invalid phoobobo account url")
	}
	return strings.TrimRight(base, "/") + "/api/account/session", overrideHost, nil
}

func phooboboUpstreamHost(publicURL string) string {
	raw := strings.TrimSpace(publicURL)
	if raw == "" {
		return phooboboAccountHost
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return phooboboAccountHost
	}
	return u.Hostname()
}

func (ctrl *V1Controller) phooboboLoginURL(r *http.Request) string {
	base := strings.TrimRight(strings.TrimSpace(ctrl.config.Phoobobo.AccountURL), "/")
	if base == "" {
		return "/"
	}
	return base + "/login?mode=" + phooboboAuthMode(r) + "&next=" + url.QueryEscape(ctrl.phooboboCallbackURL(r))
}

func phooboboAuthMode(r *http.Request) string {
	if r != nil && strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("mode")), "register") {
		return "register"
	}
	return "login"
}

func (ctrl *V1Controller) phooboboLogoutURL(r *http.Request) string {
	base := strings.TrimRight(strings.TrimSpace(ctrl.config.Phoobobo.AccountURL), "/")
	if base == "" {
		return "/"
	}
	return base + "/logout?next=" + url.QueryEscape(ctrl.phooboboReturnURL(r))
}

func (ctrl *V1Controller) phooboboReturnURL(r *http.Request) string {
	if base := phooboboPublicBase(ctrl.config.Options.Hostname); base != "" {
		return base + "/"
	}
	return phooboboRequestBase(r) + "/"
}

func (ctrl *V1Controller) phooboboCallbackURL(r *http.Request) string {
	if base := phooboboPublicBase(ctrl.config.Options.Hostname); base != "" {
		return base + "/api/v1/users/login/phoobobo"
	}
	return phooboboRequestBase(r) + "/api/v1/users/login/phoobobo"
}

func phooboboPublicBase(hostname string) string {
	hostname = strings.TrimRight(strings.TrimSpace(hostname), "/")
	if hostname == "" {
		return ""
	}
	if strings.Contains(hostname, "://") {
		return hostname
	}
	scheme := schemeHTTPS
	if phooboboLoopback(hostname) {
		scheme = "http"
	}
	return scheme + "://" + hostname
}

func phooboboRequestBase(r *http.Request) string {
	scheme := schemeHTTPS
	if r != nil && r.TLS == nil {
		proto := strings.ToLower(firstHeaderValue(r.Header.Get("X-Forwarded-Proto")))
		host := ""
		if r.Host != "" {
			host = r.Host
		}
		if proto == "http" || (proto != schemeHTTPS && phooboboLoopback(host)) {
			scheme = "http"
		}
	}
	host := ""
	if r != nil {
		host = r.Host
	}
	return scheme + "://" + host
}

func phooboboLoopback(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
