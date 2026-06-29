# Imported Capsules

Status: generated placeholder.

This directory is reserved for materialized repository-interface/context capsules imported through Gitea.

Current declared dependencies:

- `kyba`

Rules:

1. Imported capsule snapshots are generated artifacts.
2. Agents may read local files and these imported capsules.
3. Agents must not read sibling repositories directly unless the task explicitly authorizes it.
4. Capsule freshness and compatibility are checked by `kyba-ci` and surfaced by Gitea.
