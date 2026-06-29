-- Capsule impact task and draft PR fan-out contracts for the KYBa Gitea fork overlay.
-- This schema is a contract draft for the future upstream Gitea migration framework.

CREATE TABLE IF NOT EXISTS kcp_capsule_maintenance_task (
    id BIGSERIAL PRIMARY KEY,
    capsule_impact_id BIGINT NOT NULL REFERENCES kcp_capsule_impact(id) ON DELETE CASCADE,
    repository_id BIGINT NOT NULL,
    task_ref TEXT NOT NULL,
    policy TEXT NOT NULL,
    reason TEXT NOT NULL,
    blocked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (capsule_impact_id, repository_id)
);

CREATE TABLE IF NOT EXISTS kcp_capsule_draft_pr (
    id BIGSERIAL PRIMARY KEY,
    maintenance_task_id BIGINT NOT NULL REFERENCES kcp_capsule_maintenance_task(id) ON DELETE CASCADE,
    repository_id BIGINT NOT NULL,
    branch_name TEXT NOT NULL,
    title TEXT NOT NULL,
    pull_request_id BIGINT,
    generated BOOLEAN NOT NULL DEFAULT TRUE,
    state TEXT NOT NULL DEFAULT 'planned',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (maintenance_task_id, repository_id)
);

CREATE INDEX IF NOT EXISTS idx_kcp_capsule_maintenance_task_repo
    ON kcp_capsule_maintenance_task(repository_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_kcp_capsule_draft_pr_repo_state
    ON kcp_capsule_draft_pr(repository_id, state, created_at DESC);
