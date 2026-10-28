<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->

# 0001. Use the phoobobo account site for Zainaer login

## Context

Zainaer needs sign-in for the household app without a second account system. The account site is `https://account.phoobobo.top`, served by [Phoobobo/zjsy-bbs](https://github.com/Phoobobo/zjsy-bbs). The contract is `docs/INTEGRATION.md` in that repository and may later move to phoobobo-account. Zainaer reaches it only when `HBOX_PHOOBOBO_ACCOUNT_URL` is set. Operational values stay in the 「账号与备案」 section of `AGENTS.md`.

Existing Zainaer users were pre-imported into the account service with their original password hashes, so they could sign in there with the same password. That import is not a commit in this repository. Do not copy hashes, addresses, or phone numbers into this tree.

Most people who use Zainaer use only this product. The first integration labeled the button 「phoobobo 账号登录」, which described an engineering reuse they do not need to see.

The account site does not prove email ownership at registration. The first integration reused generic OIDC email migration: when no issuer and subject existed, it linked the local user with the same email. Registering someone else's email on the account site and clicking 登录 would permanently attach their Zainaer data. The same path applied to the synthetic mailbox derived from a phone number at `@users.phoobobo.top`.

## Decision

- An empty `HBOX_PHOOBOBO_ACCOUNT_URL` keeps the on-site email, password, and passkey form. Identity for a configured account is the fixed issuer `https://account.phoobobo.top` plus the account subject. Session checks send `phoobobo_sid`, then the legacy `zjsy_sid`. Guest and anonymous sessions are signed out.
- When the URL is set, signed-out 登录 and 注册 go to the account site `/login?mode=login|register&next=` the Zainaer exchange endpoint. 退出登录 clears the local session, then opens the account site `/logout?next=` back to Zainaer.
- The on-site password form remains at `/?local=1` and is not linked from the UI. A failed session check returns to `/`, not to that path.
- Visible copy is plain 登录 / 注册 / 退出登录. English is Log in / Sign up / Log out. Do not mention a shared or unified account.
- `LoginPhoobobo` links an existing local user by email only when the account session has `email_verified: true`. A missing field is false. Phone-derived addresses are never used for matching. An unverified email that already belongs to a local user does not link and does not create a second user; the browser returns to `/?login_error=email_unverified`. A new phone-only account with no conflict is still created. Other OIDC issuers keep `LoginOIDC` email migration.

## Consequences

- Account-site downtime leaves the unpublished `/?local=1` form as the operator escape hatch.
- An unverified account-site registration cannot take over an existing Zainaer user. A verified email can still be linked once the account service sets `email_verified`.
- Imported users depend on the account service keeping the original hashes. This repository does not contain those hashes.
- v0.1.21 added the account button beside the local form. v0.1.22 replaced that button with the direct 登录/注册 redirect and shared logout. v0.1.23 stopped unverified email linking.

## Record

- [Pull request #2](https://github.com/Phoobobo/zainaer/pull/2), release [v0.1.21](https://github.com/Phoobobo/zainaer/releases/tag/v0.1.21)
- [Pull request #3](https://github.com/Phoobobo/zainaer/pull/3), release [v0.1.22](https://github.com/Phoobobo/zainaer/releases/tag/v0.1.22)
- [Pull request #4](https://github.com/Phoobobo/zainaer/pull/4), release [v0.1.23](https://github.com/Phoobobo/zainaer/releases/tag/v0.1.23)
