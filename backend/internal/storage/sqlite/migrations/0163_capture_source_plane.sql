-- K4-1: consent and portable source identity only. No capture, payload bytes,
-- provider route, admission, or change_log triggers are introduced here.
-- +goose Up
-- +goose StatementBegin
CREATE TABLE capture_grants (
    grant_id TEXT NOT NULL PRIMARY KEY CHECK (length(trim(grant_id)) > 0),
    owner_id TEXT NOT NULL CHECK (length(trim(owner_id)) > 0),
    device_id TEXT NOT NULL CHECK (length(trim(device_id)) > 0),
    modality TEXT NOT NULL CHECK (modality IN ('screen_frame','ax_tree','audio_segment')),
    os_permission_kind TEXT NOT NULL CHECK (
        (modality = 'screen_frame' AND os_permission_kind = 'screen_recording') OR
        (modality = 'ax_tree' AND os_permission_kind = 'accessibility') OR
        (modality = 'audio_segment' AND os_permission_kind = 'microphone')
    ),
    os_permission_state TEXT NOT NULL CHECK (os_permission_state IN ('unknown','not_requested','granted','denied','restricted')),
    scope_json TEXT NOT NULL CHECK (json_valid(scope_json) AND json_type(scope_json) = 'object' AND length(scope_json) <= 32768),
    purpose TEXT NOT NULL CHECK (length(trim(purpose)) > 0 AND length(purpose) <= 1024),
    allowed_space_ids_json TEXT NOT NULL CHECK (json_valid(allowed_space_ids_json) AND json_type(allowed_space_ids_json) = 'array' AND length(allowed_space_ids_json) <= 8192),
    processing_route TEXT NOT NULL CHECK (processing_route = 'local'),
    disclosure_policy TEXT NOT NULL CHECK (disclosure_policy IN ('local_only','owner_review')),
    raw_retention_seconds INTEGER NOT NULL CHECK (raw_retention_seconds BETWEEN 0 AND 2592000),
    derived_retention_seconds INTEGER NOT NULL CHECK (derived_retention_seconds BETWEEN 0 AND 31536000),
    sensitivity TEXT NOT NULL CHECK (sensitivity IN ('ordinary','sensitive','restricted')),
    bystander_policy TEXT NOT NULL CHECK (bystander_policy IN ('exclude','redact_before_derive','owner_review')),
    export_behavior TEXT NOT NULL CHECK (export_behavior IN ('owner_request_only','prohibited')),
    delete_behavior TEXT NOT NULL CHECK (delete_behavior IN ('cascade_local_custody','retain_content_free_receipt')),
    state TEXT NOT NULL CHECK (state IN ('active','paused','denied','stale','failed','revoked','deleted')),
    policy_generation INTEGER NOT NULL CHECK (policy_generation >= 1),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL CHECK (updated_at >= created_at),
    paused_at TIMESTAMP,
    revoked_at TIMESTAMP,
    deleted_at TIMESTAMP,
    CHECK (state != 'active' OR os_permission_state = 'granted'),
    CHECK (state != 'paused' OR paused_at IS NOT NULL),
    CHECK (state != 'revoked' OR revoked_at IS NOT NULL),
    CHECK (state != 'deleted' OR deleted_at IS NOT NULL),
    CHECK (paused_at IS NULL OR (paused_at >= created_at AND paused_at <= updated_at)),
    CHECK (revoked_at IS NULL OR (revoked_at >= created_at AND revoked_at <= updated_at)),
    CHECK (deleted_at IS NULL OR (deleted_at >= created_at AND deleted_at <= updated_at)),
    UNIQUE (owner_id, grant_id, modality)
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX capture_grants_owner_device_state ON capture_grants(owner_id, device_id, state);
-- +goose StatementEnd

-- Hash is over the exact encrypted-blob plaintext bytes at the custody
-- boundary; only an opaque local custody reference is stored in SQLite.
-- A matching hash may occur in several separately governed grants.
-- +goose StatementBegin
CREATE TABLE source_artifacts (
    artifact_id TEXT NOT NULL PRIMARY KEY CHECK (length(trim(artifact_id)) > 0),
    owner_id TEXT NOT NULL CHECK (length(trim(owner_id)) > 0),
    grant_id TEXT NOT NULL,
    grant_generation INTEGER NOT NULL CHECK (grant_generation >= 1),
    modality TEXT NOT NULL CHECK (modality IN ('screen_frame','ax_tree','audio_segment')),
    content_kind TEXT NOT NULL CHECK (content_kind IN ('image','structured','audio')),
    mime_type TEXT NOT NULL CHECK (length(trim(mime_type)) > 0 AND length(mime_type) <= 255),
    hash_algorithm TEXT NOT NULL CHECK (hash_algorithm = 'sha256'),
    content_hash TEXT NOT NULL CHECK (length(content_hash) = 64 AND content_hash NOT GLOB '*[^0-9a-f]*'),
    blob_custody_ref TEXT NOT NULL CHECK (blob_custody_ref LIKE 'local:%' AND length(blob_custody_ref) > 6),
    provenance_kind TEXT NOT NULL CHECK (provenance_kind IN ('screen_capture','accessibility_snapshot','audio_capture')),
    provenance_ref TEXT NOT NULL CHECK (length(trim(provenance_ref)) > 0 AND length(provenance_ref) <= 1024),
    captured_at TIMESTAMP NOT NULL,
    lifecycle_state TEXT NOT NULL CHECK (lifecycle_state IN ('available','deletion_pending','deleted')),
    created_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    CHECK (lifecycle_state != 'deleted' OR deleted_at IS NOT NULL),
    CHECK ((modality = 'screen_frame' AND content_kind = 'image' AND provenance_kind = 'screen_capture') OR
           (modality = 'ax_tree' AND content_kind = 'structured' AND provenance_kind = 'accessibility_snapshot') OR
           (modality = 'audio_segment' AND content_kind = 'audio' AND provenance_kind = 'audio_capture')),
    CHECK ((modality = 'screen_frame' AND mime_type LIKE 'image/%') OR
           (modality = 'ax_tree' AND mime_type IN ('application/json','application/vnd.kennel.ax+json')) OR
           (modality = 'audio_segment' AND mime_type LIKE 'audio/%')),
    FOREIGN KEY (owner_id, grant_id, modality) REFERENCES capture_grants(owner_id, grant_id, modality),
    UNIQUE (owner_id, grant_id, artifact_id, modality, grant_generation),
    UNIQUE (owner_id, grant_id, grant_generation, hash_algorithm, content_hash)
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX source_artifacts_grant_generation ON source_artifacts(owner_id, grant_id, grant_generation, captured_at);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE UNIQUE INDEX source_artifacts_custody_ref ON source_artifacts(blob_custody_ref);
-- +goose StatementEnd

-- A segment can identify a time interval or a byte/structural offset range;
-- neither assumes OCR, image framing, nor an audio transcript.
-- +goose StatementBegin
CREATE TABLE source_segments (
    segment_id TEXT NOT NULL PRIMARY KEY CHECK (length(trim(segment_id)) > 0),
    owner_id TEXT NOT NULL CHECK (length(trim(owner_id)) > 0),
    grant_id TEXT NOT NULL,
    artifact_id TEXT NOT NULL,
    grant_generation INTEGER NOT NULL CHECK (grant_generation >= 1),
    modality TEXT NOT NULL CHECK (modality IN ('screen_frame','ax_tree','audio_segment')),
    ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
    start_offset INTEGER,
    end_offset INTEGER,
    start_at TIMESTAMP,
    end_at TIMESTAMP,
    provenance_ref TEXT NOT NULL CHECK (length(trim(provenance_ref)) > 0 AND length(provenance_ref) <= 1024),
    lifecycle_state TEXT NOT NULL CHECK (lifecycle_state IN ('available','deletion_pending','deleted')),
    created_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    CHECK ((start_offset IS NOT NULL AND end_offset IS NOT NULL AND start_offset >= 0 AND end_offset > start_offset AND start_at IS NULL AND end_at IS NULL) OR
           (start_offset IS NULL AND end_offset IS NULL AND start_at IS NOT NULL AND end_at IS NOT NULL AND end_at >= start_at)),
    CHECK (lifecycle_state != 'deleted' OR deleted_at IS NOT NULL),
    FOREIGN KEY (owner_id, grant_id, artifact_id, modality, grant_generation)
        REFERENCES source_artifacts(owner_id, grant_id, artifact_id, modality, grant_generation),
    UNIQUE (owner_id, artifact_id, ordinal)
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX source_segments_lineage ON source_segments(owner_id, grant_id, artifact_id, ordinal);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE source_segments;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE source_artifacts;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE capture_grants;
-- +goose StatementEnd
