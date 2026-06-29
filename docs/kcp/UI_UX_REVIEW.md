# KYBa KCP Repository UI/UX Review

Status: applied UI/UX correction.

## Review conclusion

KCP must be integrated into the repository interface, not presented as a separate top-level application.

Reasoning from a UI/UX perspective:

1. **User task context is repository-scoped.** Imports, exports and impact make sense only for the current repository.
2. **Agents need bounded context.** A repository-native view reinforces the rule: current repo plus materialized imported capsules, not broad sibling-repo browsing.
3. **Export file selection belongs near repository files.** Exporters should see file paths, capsule ownership and consumer targets while staying inside the exporter repo.
4. **Imported files should look like part of the repo contract.** Consumers should see materialized imports as controlled interface files, not as remote links.
5. **Native Gitea navigation reduces cognitive cost.** A `KYBa KCP` tab next to Code/Issues/Pulls is easier to discover and review than a separate platform dashboard.

## Applied UI changes

| Area | Decision |
|---|---|
| Top navbar | Removed visible KCP top-level navigation. |
| Repository header | Added native `KYBa KCP` tab. |
| Repository overview | Shows imports, exports and impact summary. |
| Imported files | Shows materialized imported capsule files in current repo context. |
| Exported files | Shows selectable exported file rows with capsule, target and mode. |
| Impact | Shows dependent repositories, KYBa tasks and draft PR plans. |

## Native URLs

```text
/{owner}/{repo}/kcp
/{owner}/{repo}/kcp/imports
/{owner}/{repo}/kcp/exports
/{owner}/{repo}/kcp/impact
```

## Next UI slice

Persistent service-backed UI should add:

- real capsule records from database;
- save/export selection persistence;
- diff between previous and current export selection;
- file tree picker instead of only table rows;
- permission-aware visibility by imported/exported capsule;
- empty/error/unavailable states;
- integration with KYBa task creation and draft PR fan-out.
