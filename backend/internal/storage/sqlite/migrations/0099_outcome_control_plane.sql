-- +goose Up
-- Durable Outcome control plane. Session activity remains an observation; task
-- lifecycle, assignments, evidence, and human decisions are authoritative here.
CREATE TABLE outcomes (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id),
  orchestrator_session_id TEXT REFERENCES sessions(id),
  title TEXT NOT NULL,
  definition TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft','planning','in_progress','blocked','completed','deleting','cleanup_blocked','deleted')),
  approved_revision INTEGER,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT
);

CREATE TABLE outcome_plan_revisions (
  outcome_id TEXT NOT NULL REFERENCES outcomes(id) ON DELETE CASCADE,
  revision INTEGER NOT NULL,
  plan_json TEXT NOT NULL,
  approved_at TEXT,
  approved_by TEXT,
  created_at TEXT NOT NULL,
  PRIMARY KEY (outcome_id, revision)
);

CREATE TABLE outcome_tasks (
  id TEXT PRIMARY KEY,
  outcome_id TEXT NOT NULL REFERENCES outcomes(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  brief TEXT NOT NULL,
  requested_harness TEXT,
  assigned_harness TEXT,
  worker_session_id TEXT REFERENCES sessions(id),
  status TEXT NOT NULL CHECK (status IN ('planned','assigned','working','on_hold','awaiting_human_decision','in_review','ready_to_merge','rework','failed','accepted')),
  hold_reason TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX outcome_tasks_outcome_idx ON outcome_tasks(outcome_id, status);
CREATE UNIQUE INDEX outcome_tasks_worker_session_idx ON outcome_tasks(worker_session_id) WHERE worker_session_id IS NOT NULL;

CREATE TABLE outcome_task_dependencies (
  task_id TEXT NOT NULL REFERENCES outcome_tasks(id) ON DELETE CASCADE,
  depends_on_task_id TEXT NOT NULL REFERENCES outcome_tasks(id) ON DELETE RESTRICT,
  PRIMARY KEY (task_id, depends_on_task_id),
  CHECK (task_id <> depends_on_task_id)
);

CREATE TABLE outcome_task_checks (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL REFERENCES outcome_tasks(id) ON DELETE CASCADE,
  statement TEXT NOT NULL,
  required INTEGER NOT NULL DEFAULT 1,
  accepted_at TEXT,
  accepted_by_session_id TEXT REFERENCES sessions(id)
);

CREATE TABLE outcome_task_evidence (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL REFERENCES outcome_tasks(id) ON DELETE CASCADE,
  check_id TEXT REFERENCES outcome_task_checks(id) ON DELETE SET NULL,
  reference TEXT NOT NULL,
  submitted_by_session_id TEXT REFERENCES sessions(id),
  submitted_at TEXT NOT NULL
);

CREATE TABLE outcome_human_decisions (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL UNIQUE REFERENCES outcome_tasks(id) ON DELETE CASCADE,
  question TEXT NOT NULL,
  context TEXT NOT NULL,
  options_json TEXT NOT NULL,
  recommendation TEXT NOT NULL,
  resume_target TEXT NOT NULL,
  answer TEXT,
  resolved_at TEXT
);

CREATE TABLE outcome_task_transitions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id TEXT NOT NULL REFERENCES outcome_tasks(id) ON DELETE CASCADE,
  from_status TEXT,
  to_status TEXT NOT NULL,
  reason TEXT NOT NULL,
  changed_at TEXT NOT NULL
);

-- Tombstones deliberately exclude prompt/transcript content. They allow
-- interrupted native-session cleanup to retry without resurrecting an Outcome.
CREATE TABLE outcome_cleanup_audit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  outcome_id TEXT NOT NULL,
  session_id TEXT,
  native_session_id TEXT,
  harness TEXT,
  state TEXT NOT NULL,
  detail TEXT,
  recorded_at TEXT NOT NULL
);

-- +goose Down
DROP TABLE outcome_cleanup_audit;
DROP TABLE outcome_task_transitions;
DROP TABLE outcome_human_decisions;
DROP TABLE outcome_task_evidence;
DROP TABLE outcome_task_checks;
DROP TABLE outcome_task_dependencies;
DROP TABLE outcome_tasks;
DROP TABLE outcome_plan_revisions;
DROP TABLE outcomes;
