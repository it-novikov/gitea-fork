# KYBa KCP DB-backed Model

Status: active implementation note.
Owner: Gitea Fork maintainers / KYBa Platform.
Last reviewed: 2026-06-30.

## Tables

Migration `v331` creates the original KCP tables:

```text
kcp_repository_interface
kcp_repository_interface_import
kcp_capsule_impact_task
kcp_archive_export_run
```

Migration `v332` adds P1 persistence and permission tables:

```text
kcp_repository_interface_file
kcp_permission_grant
```

## Repository-native sync

The repository KCP pages derive the current Git tree, then persist repository-scoped state:

- selected export files -> `kcp_repository_interface_file`;
- repository interface manifest -> `kcp_repository_interface`;
- materialized imported capsules observed in `.kyba/imported-capsules/**` -> `kcp_repository_interface_import`;
- impact rows shown in the UI -> `kcp_capsule_impact_task`.

This gives the API and future task automation a durable model without requiring agents to scan sibling repositories.

## Current non-goals

- Full admin UI for KCP permission grants.
- Automatic draft PR creation.
- Cross-repo fan-out tasks.

Those are next service slices.
