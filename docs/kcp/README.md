# KYBa KCP in Gitea Fork

Status: integrated fork documentation.

KYBa KCP extends Gitea with repository interfaces, context capsules, impact analysis and reproducible archive export for KYBa multirepo work.

## Web UI

KCP is embedded into the normal repository interface as the `KYBa KCP` repository tab.

Repository-scoped pages:

- `/{owner}/{repo}/kcp`
- `/{owner}/{repo}/kcp/imports`
- `/{owner}/{repo}/kcp/exports`
- `/{owner}/{repo}/kcp/impact`

The main Gitea top navbar does not expose KCP as a separate app. This is intentional: imported/exported files and impact analysis belong in repository context.

## Repository behavior

Exporter repositories can see and select exported files on `/{owner}/{repo}/kcp/exports`.

Consumer repositories can see imported, materialized capsule files on `/{owner}/{repo}/kcp/imports`.

Impact records appear on `/{owner}/{repo}/kcp/impact` and show dependent repositories, KYBa maintenance tasks and draft PR plans.

## Identity Service authentication

User-facing login, registration and account recovery can be delegated to KYBa Identity Service through `[kyba.identity]`.

When enabled, Gitea renders KYBa Identity pages for:

- `/user/login`;
- `/user/identity/challenge`;
- `/user/sign_up`;
- `/user/forgot_password`.

The Identity Service verifies the principal, and Gitea creates or syncs a local shadow user for sessions and repository permissions.

See `docs/kcp/IDENTITY_SERVICE_AUTH.md`.

## API

The integrated fork exposes admin API draft endpoints under `/api/v1/kcp`:

- `/api/v1/kcp/capsules`
- `/api/v1/kcp/imports`
- `/api/v1/kcp/impact`
- `/api/v1/kcp/export-plan`

## Database

The fork adds Gitea migration `v331` with KCP tables for repository interfaces, imports, impact tasks and archive export runs.

## Current implementation status

This is a deployable Gitea fork source tree with KCP packages, repository-native routes, templates, migration and API draft. The current route handlers expose deterministic domain data until the next slice wires the service layer to persistent records and KYBa task creation.
