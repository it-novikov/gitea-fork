-- Repository interface/capsule registry foundation for the KYBa Gitea fork overlay.
-- This is a schema contract draft; production integration must be ported into the selected
-- upstream Gitea migration framework.

CREATE TABLE IF NOT EXISTS kcp_capsule_export (
    id BIGSERIAL PRIMARY KEY,
    capsule_id TEXT NOT NULL UNIQUE,
    owner_repository_id BIGINT NOT NULL,
    kind TEXT NOT NULL,
    version TEXT NOT NULL,
    visibility TEXT NOT NULL,
    manifest_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS kcp_capsule_import (
    id BIGSERIAL PRIMARY KEY,
    capsule_export_id BIGINT NOT NULL REFERENCES kcp_capsule_export(id) ON DELETE CASCADE,
    consumer_repository_id BIGINT NOT NULL,
    required_version TEXT NOT NULL,
    materialized_revision TEXT,
    freshness_status TEXT NOT NULL DEFAULT 'unknown',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (capsule_export_id, consumer_repository_id)
);

CREATE TABLE IF NOT EXISTS kcp_capsule_impact (
    id BIGSERIAL PRIMARY KEY,
    capsule_export_id BIGINT NOT NULL REFERENCES kcp_capsule_export(id) ON DELETE CASCADE,
    change_sha TEXT NOT NULL,
    compatible BOOLEAN NOT NULL,
    impact_report_json JSONB NOT NULL,
    kyba_task_ref TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_kcp_capsule_import_consumer
    ON kcp_capsule_import (consumer_repository_id);

CREATE INDEX IF NOT EXISTS idx_kcp_capsule_impact_export
    ON kcp_capsule_impact (capsule_export_id, created_at DESC);
