-- name: InsertCaptureGrant :exec
INSERT INTO capture_grants(grant_id,owner_id,device_id,modality,os_permission_kind,os_permission_state,scope_json,purpose,allowed_space_ids_json,processing_route,disclosure_policy,raw_retention_seconds,derived_retention_seconds,sensitivity,bystander_policy,export_behavior,delete_behavior,state,policy_generation,created_at,updated_at,paused_at,revoked_at,deleted_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);

-- name: GetCaptureGrant :one
SELECT * FROM capture_grants WHERE owner_id=? AND grant_id=?;

-- name: UpdateCaptureGrantPolicy :execrows
UPDATE capture_grants SET
    os_permission_state=sqlc.arg(os_permission_state),
    scope_json=sqlc.arg(scope_json), purpose=sqlc.arg(purpose),
    allowed_space_ids_json=sqlc.arg(allowed_space_ids_json),
    disclosure_policy=sqlc.arg(disclosure_policy),
    raw_retention_seconds=sqlc.arg(raw_retention_seconds),
    derived_retention_seconds=sqlc.arg(derived_retention_seconds),
    sensitivity=sqlc.arg(sensitivity), bystander_policy=sqlc.arg(bystander_policy),
    export_behavior=sqlc.arg(export_behavior), delete_behavior=sqlc.arg(delete_behavior),
    state=sqlc.arg(state), policy_generation=policy_generation+1,
    updated_at=sqlc.arg(updated_at), paused_at=sqlc.arg(paused_at),
    revoked_at=sqlc.arg(revoked_at), deleted_at=sqlc.arg(deleted_at)
WHERE owner_id=sqlc.arg(owner_id) AND grant_id=sqlc.arg(grant_id)
  AND modality=sqlc.arg(modality) AND device_id=sqlc.arg(device_id)
  AND policy_generation=sqlc.arg(expected_generation)
  AND state!='deleted' AND (state!='revoked' OR sqlc.arg(state)='deleted');

-- name: InsertSourceArtifactIfGrantCurrent :execrows
INSERT INTO source_artifacts(artifact_id,owner_id,grant_id,grant_generation,modality,content_kind,mime_type,hash_algorithm,content_hash,blob_custody_ref,provenance_kind,provenance_ref,captured_at,lifecycle_state,created_at,deleted_at)
SELECT sqlc.arg(artifact_id),sqlc.arg(owner_id),sqlc.arg(grant_id),sqlc.arg(grant_generation),sqlc.arg(modality),sqlc.arg(content_kind),sqlc.arg(mime_type),sqlc.arg(hash_algorithm),sqlc.arg(content_hash),sqlc.arg(blob_custody_ref),sqlc.arg(provenance_kind),sqlc.arg(provenance_ref),sqlc.arg(captured_at),sqlc.arg(lifecycle_state),sqlc.arg(created_at),sqlc.arg(deleted_at)
FROM capture_grants
WHERE owner_id=sqlc.arg(owner_id) AND grant_id=sqlc.arg(grant_id)
  AND modality=sqlc.arg(modality) AND policy_generation=sqlc.arg(grant_generation)
  AND state='active' AND os_permission_state='granted';

-- name: GetSourceArtifact :one
SELECT a.* FROM source_artifacts a JOIN capture_grants g
  ON g.owner_id=a.owner_id AND g.grant_id=a.grant_id AND g.modality=a.modality
WHERE a.owner_id=? AND a.artifact_id=? AND a.lifecycle_state='available'
  AND g.state='active' AND g.os_permission_state='granted'
  AND g.policy_generation=a.grant_generation;

-- name: InsertSourceSegmentIfGrantCurrent :execrows
INSERT INTO source_segments(segment_id,owner_id,grant_id,artifact_id,grant_generation,modality,ordinal,start_offset,end_offset,start_at,end_at,provenance_ref,lifecycle_state,created_at,deleted_at)
SELECT sqlc.arg(segment_id),sqlc.arg(owner_id),sqlc.arg(grant_id),sqlc.arg(artifact_id),sqlc.arg(grant_generation),sqlc.arg(modality),sqlc.arg(ordinal),sqlc.arg(start_offset),sqlc.arg(end_offset),sqlc.arg(start_at),sqlc.arg(end_at),sqlc.arg(provenance_ref),sqlc.arg(lifecycle_state),sqlc.arg(created_at),sqlc.arg(deleted_at)
FROM capture_grants g JOIN source_artifacts a
  ON a.owner_id=g.owner_id AND a.grant_id=g.grant_id AND a.modality=g.modality
  AND a.artifact_id=sqlc.arg(artifact_id) AND a.grant_generation=sqlc.arg(grant_generation)
WHERE g.owner_id=sqlc.arg(owner_id) AND g.grant_id=sqlc.arg(grant_id)
  AND g.modality=sqlc.arg(modality) AND g.policy_generation=sqlc.arg(grant_generation)
  AND g.state='active' AND g.os_permission_state='granted' AND a.lifecycle_state='available';

-- name: GetSourceSegment :one
SELECT s.* FROM source_segments s JOIN source_artifacts a
  ON a.owner_id=s.owner_id AND a.grant_id=s.grant_id AND a.artifact_id=s.artifact_id
  AND a.modality=s.modality AND a.grant_generation=s.grant_generation
  JOIN capture_grants g ON g.owner_id=s.owner_id AND g.grant_id=s.grant_id AND g.modality=s.modality
WHERE s.owner_id=? AND s.segment_id=? AND s.lifecycle_state='available'
  AND a.lifecycle_state='available' AND g.state='active'
  AND g.os_permission_state='granted' AND g.policy_generation=s.grant_generation;
