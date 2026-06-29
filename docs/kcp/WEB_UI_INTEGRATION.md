# KYBa KCP Native Repository UI Integration

Status: repository-native UI scaffold.

## Purpose

KYBa KCP must look and behave like a native Gitea repository capability, not like a separate application bolted onto Gitea.

The primary user journey is always repository-scoped:

```text
Repository -> KYBa KCP tab -> Overview / Imported files / Exported files / Impact
```

## Repository tab

The repository header includes a normal Gitea tab:

```text
Code | Issues | Pull Requests | Actions | Packages | Projects | Releases | Wiki | Activity | KYBa KCP | Settings
```

The tab is rendered from `templates/repo/header.tmpl` and routes to:

```text
/{owner}/{repo}/kcp
```

## Repository-native pages

| Page | URL | Purpose |
|---|---|---|
| Overview | `/{owner}/{repo}/kcp` | Repository-scoped summary of imports, exports and impact. |
| Imported files | `/{owner}/{repo}/kcp/imports` | Materialized capsule files visible to this repository. |
| Exported files | `/{owner}/{repo}/kcp/exports` | Files exported by this repository with selectable file list. |
| Impact | `/{owner}/{repo}/kcp/impact` | Dependent repositories, KYBa tasks and draft PR plans affected by capsule changes. |

## UI/UX decision

KCP is no longer exposed through the main top navbar as a separate top-level web app.

Reason:

- imports and exports are meaningful only in repository context;
- agents should read current repo + materialized imports, not unrelated platform pages;
- exporters should select files while looking at the exporter repository;
- impact should be scoped to the current repo and its dependents.

A future admin/global overview may exist, but it must not replace the repository-native workflow.

## Exporter repository behavior

Exporter repositories show:

- exported capsule ID;
- selected files;
- target repo/consumer;
- file mode: contract, context, generated, validation, interface;
- checkbox selection for export preview.

The current scaffold accepts the selection form and proves the request contract. Persistent storage is handled by the KCP repository-interface service/model integration slice.

## Consumer repository behavior

Consumer repositories show imported files as materialized local snapshots:

```text
.kyba/imported-capsules/<capsule-id>/...
```

This is intentional. It avoids broad cross-repository reads and keeps agent context bounded.

## Acceptance criteria

- `/{owner}/{repo}/kcp` renders inside normal repository shell.
- `/{owner}/{repo}/kcp/imports` shows materialized imported files.
- `/{owner}/{repo}/kcp/exports` shows selectable exported files.
- `/{owner}/{repo}/kcp/impact` shows dependent maintenance records.
- The main global navbar does not expose KCP as a separate application.
- Repository tab active state uses `PageIsRepoKCP`.
