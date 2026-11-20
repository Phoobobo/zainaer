<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->

# Private assistant conversation snapshots

## Context

The assistant had no persistent conversation model. Users need to resume conversations after refresh without replaying item operations. Shared account sign-in may link a verified identity to an existing local user; history ownership must continue to follow the authenticated local user and collection, rather than an email, phone number or model-supplied identity.

## Decision

Store private conversation snapshots under the existing AI storage tree, scoped by server-authenticated collection and user IDs. Use atomic file replacement and optimistic revisions to prevent stale tabs overwriting newer history. Keep the existing chat and explicit confirmation endpoints unchanged. Loading a snapshot restores display data and model context, but disables pending confirmation cards and never invokes tools. Switching cancels inference and local recording and discards their late results.

## Consequences

No database migration or external service is needed, and existing full-data backups include conversations. Old ephemeral conversations cannot be recovered. Snapshots have a transport size limit and are rewritten on save; large histories may eventually warrant an indexed database store. Conflicts require reloading the saved conversation rather than silently replacing another tab's data. Identity linking and verified-email requirements remain owned by the account authentication flow and are not bypassed by history APIs.
