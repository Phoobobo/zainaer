<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->

# 0003. Collection privilege tiers and private-to-self items

## Context

Collection membership previously had only `owner` vs `user`. The product needs a second tier for “privileged” members who can see internal content and each other’s privilege labels, without inventing a full RBAC system. Privileged members also need a per-item “visible only to myself” control that hides the item from everyone else, including other privileged members. Ordinary members must not learn who is privileged.

## Decision

- Keep existing `user_groups.role` (`owner` / `user`) for ownership-only actions (model config, wipe, exports, privilege grants).
- Add `user_groups.privileged` boolean. Owners are always treated as privileged in authorization (`IsPrivilegedOf`); invited members start ordinary; only owners can grant or revoke the flag via `PUT /groups/members/{id}/privileged`.
- Privileged-only item fields: `staffNotes` (shared among privileged members) and `privateToSelf` (binds `private_to_user_id` to the acting user). Ordinary members never receive these fields; writes return 403.
- Queries and gets filter by visibility: public items, or private-to-self items owned by the viewer. Locations cannot be private-to-self.
- Members list returns `GroupMembersOut` with `selfPrivileged` / `selfIsOwner`. Per-member `privileged` / `isOwner` are included only when the viewer is privileged.

## Consequences

- Privilege is orthogonal to ownership: a privileged non-owner can edit staff notes and private-to-self, but cannot grant privilege or configure models.
- Private-to-self items are invisible to AI search/tools for other users in the same collection.
- CSV export does not add staff notes or private-to-self columns in this change.
- Smallest content surface for privileged-shared data remains item/location `staffNotes`; broader privileged content types can reuse `IsPrivilegedOf`.
