# KYBa Gitea Fork Upstream Policy

This repository is a full Gitea source tree with KYBa KCP additions.

## Upstream boundary

- Upstream Gitea source remains recognizable and should be kept rebasing-friendly.
- KYBa code uses explicit `kcp` packages, routes, templates, migrations and docs.
- Avoid modifying unrelated upstream behavior unless a KCP feature requires it.

## KYBa additions

- `modules/kcp/*`
- `services/kcp/*`
- `routers/web/repo/kcp.go`
- `routers/api/v1/kcp/*`
- `templates/repo/kcp/*`
- `models/migrations/v1_26/v331.go`
- `docs/kcp/*`
- `openapi/kcp-*.yaml`
- `.kyba/*`

## Rebase rule

When updating upstream Gitea:

1. reapply KCP namespace packages;
2. recheck repository-native web route integration in `routers/web/web.go`;
3. recheck repository tab integration in `templates/repo/header.tmpl`;
4. recheck API route integration in `routers/api/v1/api.go`;
5. confirm migration number does not conflict;
6. run the validation commands from `VALIDATION.md`.
