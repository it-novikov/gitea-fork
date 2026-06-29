# KYBa KCP Validation Notes

Status: repository-native KCP validation.

## Required checks

```bash
go test ./...
make build
make frontend
```

## Repository UI smoke

Start the Gitea fork, open an existing repository and confirm:

- the repository header shows a `KYBa KCP` tab;
- `/{owner}/{repo}/kcp` renders overview cards;
- `/{owner}/{repo}/kcp/imports` shows imported materialized capsule files;
- `/{owner}/{repo}/kcp/exports` shows exported files with checkboxes;
- `/{owner}/{repo}/kcp/impact` shows dependent maintenance records.

## Owner acceptance

Before deleting the Gitea KCP code from the shared KYBa monorepo, record:

- archive uploaded to `gitea-fork`;
- `go test ./...` passed in `gitea-fork`;
- repository-native KCP UI is visible;
- exported/imported files are visible in repository context;
- owner approves removal from shared repo.
