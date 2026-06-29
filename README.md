# Gitea KYBa Fork

KYBa extends Gitea with a platform layer for repository interfaces and context capsules.

Added capabilities:

- Native repository tab `KYBa KCP` next to Code/Issues/Pulls/Actions.
- Repository-scoped imported files view: materialized capsule files are visible inside the consuming repository.
- Repository-scoped exported files view: exporter repositories can see and select files for export.
- Repository-scoped impact view: dependent repositories, KYBa maintenance tasks and draft PR plans are shown in the repository context.
- KYBa Identity Service delegated authentication for login, registration, challenge verification and account recovery.
- `/api/v1/kcp/*` admin API draft endpoints for automation and future UI persistence.
- Database migration for KCP repository interface, import, impact and export-run tables.
- KYBa module/service/router/template packages integrated into the Gitea source tree.

This fork keeps upstream Gitea behavior and embeds KYBa repository-interface workflows into the normal repository UI instead of forcing users into a separate app.

See `docs/kcp/README.md`, `docs/kcp/IDENTITY_SERVICE_AUTH.md` and `UPSTREAM_POLICY.md`.
