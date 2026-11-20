<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->

# 0003. Collection privilege tiers and private-to-self items

## Context

Collection membership previously had only `owner` vs `user`. The product needs a second tier for “privileged” members who can see internal content and each other’s privilege labels, without inventing a full RBAC system. Privileged members also need a per-item “visible only to myself” control that hides the item from everyone else, including other privileged members. Ordinary members must not learn who is privileged.

## Decision

- Keep existing `user_groups.role` (`owner` / `user`) for ownership-only actions (model config, wipe, exports, privilege grants).
- Add `user_groups.privileged` boolean. The collection creator/owner is privileged by default (`IsPrivilegedOf` = role owner OR privileged flag). Invited members start ordinary. Only the collection owner may grant or revoke privilege on other members via `PUT /groups/members/{id}/privileged` (`IsOwnerOf` gate). Privileged non-owners cannot grant privilege.
- Privileged-only shared field: `staffNotes` on both items and locations. Privileged-only per-item control: `privateToSelf` (binds `private_to_user_id` to the acting user). Ordinary members never receive these fields; writes return 403. Locations do not support `privateToSelf`.
- On privilege demotion, leave `privateToSelf` / `private_to_user_id` as-is (do not auto-clear or reassign). The item remains visible only to that user; they lose the privilege UI for new staff notes / new private flags while demoted.
- Queries and gets filter by visibility: public entities, or private-to-self items owned by the viewer.
- Members list returns `GroupMembersOut` with `selfPrivileged` / `selfIsOwner`. Per-member `privileged` / `isOwner` are included only when the viewer is privileged.

## Consequences

- Privilege is orthogonal to ownership: a privileged non-owner can edit staff notes and private-to-self, but cannot grant privilege or configure models.
- Demotion does not orphan or republish private-to-self items; only that member still sees them.
- Private-to-self items are invisible to AI search/tools for other users in the same collection.
- CSV export does not add staff notes or private-to-self columns in this change.
- Smallest privileged-shared content surface is item and location `staffNotes`; broader privileged content types can reuse `IsPrivilegedOf`.
