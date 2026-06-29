-- Archive export run foundation for the KYBa Gitea fork overlay.
-- This schema stores validated export plans and generated archive metadata.

CREATE TABLE IF NOT EXISTS kcp_archive_export_run (
    id BIGSERIAL PRIMARY KEY,
    manifest_json JSONB NOT NULL,
    import_order_json JSONB NOT NULL,
    status TEXT NOT NULL,
    assigned_files INTEGER NOT NULL DEFAULT 0,
    ambiguous_files INTEGER NOT NULL DEFAULT 0,
    unassigned_files INTEGER NOT NULL DEFAULT 0,
    report_json JSONB NOT NULL,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS kcp_archive_export_artifact (
    id BIGSERIAL PRIMARY KEY,
    export_run_id BIGINT NOT NULL REFERENCES kcp_archive_export_run(id) ON DELETE CASCADE,
    target_repo TEXT NOT NULL,
    archive_name TEXT NOT NULL,
    archive_sha256 TEXT NOT NULL,
    file_count INTEGER NOT NULL,
    generated_file_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (export_run_id, target_repo)
);

CREATE INDEX IF NOT EXISTS idx_kcp_archive_export_run_status
    ON kcp_archive_export_run (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_kcp_archive_export_artifact_run
    ON kcp_archive_export_artifact (export_run_id);
