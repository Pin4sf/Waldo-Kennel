package sqlite

import (
	"database/sql"
	"fmt"
)

// changeLogWriters lists every CDC trigger the shipped schema attaches to
// tables that write change_log, in their latest migrated form. Migration 0099
// rebuilds the checked change_log relation and must detach these writers
// first; restoring them inside that same SQL file breaks degraded profiles
// whose skipped migrations have not physically created every subject table.
//
// restoreChangeLogWriters therefore runs from migrate() after goose completes:
// it recreates each writer only when the trigger is absent AND its subject
// table exists, healing clean databases, upgraded AO-derived databases, and
// burned/skipped-migration profiles alike. Bodies must be kept verbatim with
// their defining migration.
var changeLogWriters = []struct {
	name string
	// table is the writer's direct subject table.
	table string
	// deps lists every additional table the trigger body touches. The writer
	// is restored only when the subject table and every dependency physically
	// exist, so partially migrated profiles never carry half-built triggers.
	deps []string
	sql  string
}{
	{
		name:  "agent_switches_cdc_insert",
		table: "agent_switches",
		sql:   "CREATE TRIGGER agent_switches_cdc_insert\nAFTER INSERT ON agent_switches\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT project_id FROM sessions WHERE id = NEW.session_id),\n        NEW.session_id, 'session_updated', json_object('id', NEW.session_id), NEW.updated_at\n    );\nEND;",
	},
	{
		name:  "agent_switches_cdc_update",
		table: "agent_switches",
		sql:   "CREATE TRIGGER agent_switches_cdc_update\nAFTER UPDATE ON agent_switches\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT project_id FROM sessions WHERE id = NEW.session_id),\n        NEW.session_id, 'session_updated', json_object('id', NEW.session_id), NEW.updated_at\n    );\nEND;",
	},
	{
		name:  "conversation_activities_cdc_insert",
		table: "conversation_activities",
		sql:   "CREATE TRIGGER conversation_activities_cdc_insert\nAFTER INSERT ON conversation_activities\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    SELECT s.project_id, s.id, 'session_updated',\n\t\t   json_object('id', s.id, 'sessionId', s.id, 'conversationId', c.id,\n\t\t\t\t\t   'activity', s.activity_state,\n                       'isTerminated', json(CASE WHEN s.is_terminated THEN 'true' ELSE 'false' END)),\n           NEW.updated_at\n    FROM conversations c\n    JOIN sessions s ON s.id = c.current_session_id\n    WHERE c.id = NEW.conversation_id;\nEND;",
	},
	{
		name:  "conversation_activities_cdc_update",
		table: "conversation_activities",
		sql:   "CREATE TRIGGER conversation_activities_cdc_update\nAFTER UPDATE ON conversation_activities\nWHEN OLD.revision <> NEW.revision\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    SELECT s.project_id, s.id, 'session_updated',\n\t\t   json_object('id', s.id, 'sessionId', s.id, 'conversationId', c.id,\n\t\t\t\t\t   'activity', s.activity_state,\n                       'isTerminated', json(CASE WHEN s.is_terminated THEN 'true' ELSE 'false' END)),\n           NEW.updated_at\n    FROM conversations c\n    JOIN sessions s ON s.id = c.current_session_id\n    WHERE c.id = NEW.conversation_id;\nEND;",
	},
	{
		name:  "conversation_messages_cdc_insert",
		table: "conversation_messages",
		sql:   "CREATE TRIGGER conversation_messages_cdc_insert\nAFTER INSERT ON conversation_messages\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    SELECT s.project_id, s.id, 'session_updated',\n\t\t   json_object('id', s.id, 'sessionId', s.id, 'conversationId', NEW.conversation_id,\n\t\t\t\t\t   'activity', s.activity_state,\n                       'isTerminated', json(CASE WHEN s.is_terminated THEN 'true' ELSE 'false' END)),\n           NEW.updated_at\n    FROM conversations c\n\tJOIN sessions s ON s.id = c.current_session_id\n    WHERE c.id = NEW.conversation_id;\nEND;",
	},
	{
		name:  "conversation_messages_cdc_update",
		table: "conversation_messages",
		sql:   "CREATE TRIGGER conversation_messages_cdc_update\nAFTER UPDATE ON conversation_messages\nWHEN OLD.revision <> NEW.revision\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    SELECT s.project_id, s.id, 'session_updated',\n           json_object('id', s.id, 'sessionId', s.id, 'conversationId', c.id,\n\t\t\t\t\t   'activity', s.activity_state,\n                       'isTerminated', json(CASE WHEN s.is_terminated THEN 'true' ELSE 'false' END)),\n           NEW.updated_at\n    FROM conversations c\n    JOIN sessions s ON s.id = c.current_session_id\n    WHERE c.id = NEW.conversation_id;\nEND;",
	},
	{
		name:  "conversation_turns_cdc_update",
		table: "conversation_turns",
		sql:   "CREATE TRIGGER conversation_turns_cdc_update\nAFTER UPDATE ON conversation_turns\nWHEN OLD.state <> NEW.state\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    SELECT s.project_id, s.id, 'session_updated',\n\t\t   json_object('id', s.id, 'sessionId', s.id, 'conversationId', NEW.conversation_id,\n\t\t\t\t\t   'activity', s.activity_state,\n                       'isTerminated', json(CASE WHEN s.is_terminated THEN 'true' ELSE 'false' END)),\n           COALESCE(NEW.completed_at, NEW.started_at, NEW.requested_at)\n    FROM sessions s\n    WHERE s.id = NEW.handled_by_session_id;\nEND;",
	},
	{
		name:  "pr_cdc_insert",
		table: "pr",
		sql:   "CREATE TRIGGER pr_cdc_insert\nAFTER INSERT ON pr\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES ((SELECT project_id FROM sessions WHERE id = NEW.session_id), NEW.session_id, 'pr_created',\n        json_object('url', NEW.url, 'session', NEW.session_id, 'state', NEW.pr_state,\n                    'ci', NEW.ci_state, 'review', NEW.review_decision, 'mergeability', NEW.mergeability),\n        NEW.updated_at);\nEND;",
	},
	{
		name:  "pr_cdc_update",
		table: "pr",
		sql:   "CREATE TRIGGER pr_cdc_update\nAFTER UPDATE ON pr\nWHEN OLD.pr_state <> NEW.pr_state\n    OR OLD.ci_state <> NEW.ci_state\n    OR OLD.review_decision <> NEW.review_decision\n    OR OLD.mergeability <> NEW.mergeability\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES ((SELECT project_id FROM sessions WHERE id = NEW.session_id), NEW.session_id, 'pr_updated',\n        json_object('url', NEW.url, 'session', NEW.session_id, 'state', NEW.pr_state,\n                    'ci', NEW.ci_state, 'review', NEW.review_decision, 'mergeability', NEW.mergeability),\n        NEW.updated_at);\nEND;",
	},
	{
		name:  "pr_checks_cdc_insert",
		table: "pr_checks",
		sql:   "CREATE TRIGGER pr_checks_cdc_insert\nAFTER INSERT ON pr_checks\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT s.project_id FROM pr p JOIN sessions s ON s.id = p.session_id WHERE p.url = NEW.pr_url),\n        (SELECT session_id FROM pr WHERE url = NEW.pr_url),\n        'pr_check_recorded',\n        json_object('pr', NEW.pr_url, 'name', NEW.name, 'commit', NEW.commit_hash, 'status', NEW.status),\n        NEW.created_at);\nEND;",
	},
	{
		name:  "pr_checks_cdc_update",
		table: "pr_checks",
		sql:   "CREATE TRIGGER pr_checks_cdc_update\nAFTER UPDATE ON pr_checks\nWHEN OLD.status <> NEW.status\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT s.project_id FROM pr p JOIN sessions s ON s.id = p.session_id WHERE p.url = NEW.pr_url),\n        (SELECT session_id FROM pr WHERE url = NEW.pr_url),\n        'pr_check_recorded',\n        json_object('pr', NEW.pr_url, 'name', NEW.name, 'commit', NEW.commit_hash, 'status', NEW.status),\n        datetime('now'));\nEND;",
	},
	{
		name:  "pr_review_threads_cdc_insert",
		table: "pr_review_threads",
		sql:   "CREATE TRIGGER pr_review_threads_cdc_insert\nAFTER INSERT ON pr_review_threads\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT s.project_id FROM pr p JOIN sessions s ON s.id = p.session_id WHERE p.url = NEW.pr_url),\n        (SELECT session_id FROM pr WHERE url = NEW.pr_url),\n        'pr_review_thread_added',\n        json_object(\n            'pr', NEW.pr_url,\n            'thread', NEW.thread_id,\n            'path', NEW.path,\n            'line', NEW.line,\n            'resolved', json(CASE WHEN NEW.resolved THEN 'true' ELSE 'false' END),\n            'isBot', json(CASE WHEN NEW.is_bot THEN 'true' ELSE 'false' END)\n        ),\n        NEW.updated_at);\nEND;",
	},
	{
		name:  "pr_review_threads_cdc_update",
		table: "pr_review_threads",
		sql:   "CREATE TRIGGER pr_review_threads_cdc_update\nAFTER UPDATE ON pr_review_threads\nWHEN OLD.resolved <> NEW.resolved\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT s.project_id FROM pr p JOIN sessions s ON s.id = p.session_id WHERE p.url = NEW.pr_url),\n        (SELECT session_id FROM pr WHERE url = NEW.pr_url),\n        'pr_review_thread_resolved',\n        json_object(\n            'pr', NEW.pr_url,\n            'thread', NEW.thread_id,\n            'path', NEW.path,\n            'line', NEW.line,\n            'resolved', json(CASE WHEN NEW.resolved THEN 'true' ELSE 'false' END)\n        ),\n        NEW.updated_at);\nEND;",
	},
	{
		name:  "pr_session_cdc_update",
		table: "pr",
		sql:   "CREATE TRIGGER pr_session_cdc_update\nAFTER UPDATE ON pr\nWHEN OLD.session_id <> NEW.session_id\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT project_id FROM sessions WHERE id = NEW.session_id),\n        NEW.session_id,\n        'pr_session_changed',\n        json_object(\n            'url', NEW.url,\n            'fromSession', OLD.session_id,\n            'toSession', NEW.session_id),\n        NEW.updated_at);\nEND;",
	},
	{
		name:  "session_cleanup_facts_cdc_insert",
		table: "session_cleanup_facts",
		sql:   "CREATE TRIGGER session_cleanup_facts_cdc_insert\nAFTER INSERT ON session_cleanup_facts\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES ((SELECT project_id FROM sessions WHERE id = NEW.session_id), NEW.session_id, 'session_updated',\n        json_object('id', NEW.session_id),\n        datetime('now'));\nEND;",
	},
	{
		name:  "session_cleanup_facts_cdc_update",
		table: "session_cleanup_facts",
		sql:   "CREATE TRIGGER session_cleanup_facts_cdc_update\nAFTER UPDATE ON session_cleanup_facts\nWHEN OLD.workspace_disposition <> NEW.workspace_disposition\n    OR (OLD.runtime_released_at IS NULL) <> (NEW.runtime_released_at IS NULL)\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES ((SELECT project_id FROM sessions WHERE id = NEW.session_id), NEW.session_id, 'session_updated',\n        json_object('id', NEW.session_id),\n        datetime('now'));\nEND;",
	},
	{
		name:  "session_interface_transitions_cdc_insert",
		table: "session_interface_transitions",
		sql:   "CREATE TRIGGER session_interface_transitions_cdc_insert\nAFTER INSERT ON session_interface_transitions\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    SELECT s.project_id, s.id, 'session_updated',\n           json_object('id', s.id, 'sessionId', s.id,\n                       'interfaceTransitionId', NEW.id,\n                       'interfaceTransitionPhase', NEW.phase,\n                       'activity', s.activity_state,\n                       'isTerminated', json(CASE WHEN s.is_terminated THEN 'true' ELSE 'false' END)),\n           NEW.updated_at\n    FROM sessions s WHERE s.id = NEW.session_id;\nEND;",
	},
	{
		name:  "session_interface_transitions_cdc_update",
		table: "session_interface_transitions",
		sql:   "CREATE TRIGGER session_interface_transitions_cdc_update\nAFTER UPDATE ON session_interface_transitions\nWHEN OLD.phase <> NEW.phase\n    OR OLD.error_code <> NEW.error_code\n    OR OLD.error_detail <> NEW.error_detail\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    SELECT s.project_id, s.id, 'session_updated',\n           json_object('id', s.id, 'sessionId', s.id,\n                       'interfaceTransitionId', NEW.id,\n                       'interfaceTransitionPhase', NEW.phase,\n                       'activity', s.activity_state,\n                       'isTerminated', json(CASE WHEN s.is_terminated THEN 'true' ELSE 'false' END)),\n           NEW.updated_at\n    FROM sessions s WHERE s.id = NEW.session_id;\nEND;",
	},
	{
		name:  "sessions_cdc_insert",
		table: "sessions",
		sql:   "CREATE TRIGGER sessions_cdc_insert\nAFTER INSERT ON sessions\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (NEW.project_id, NEW.id, 'session_created',\n        json_object('id', NEW.id, 'activity', NEW.activity_state, 'isTerminated', json(CASE WHEN NEW.is_terminated THEN 'true' ELSE 'false' END)),\n        NEW.updated_at);\nEND;",
	},
	{
		name:  "sessions_cdc_update",
		table: "sessions",
		sql:   "CREATE TRIGGER sessions_cdc_update\nAFTER UPDATE ON sessions\nWHEN OLD.activity_state <> NEW.activity_state\n    OR OLD.is_terminated <> NEW.is_terminated\n    OR (OLD.first_signal_at IS NULL AND NEW.first_signal_at IS NOT NULL)\n    OR OLD.preview_url <> NEW.preview_url\n    OR OLD.preview_revision <> NEW.preview_revision\n    OR OLD.display_name <> NEW.display_name\n    OR OLD.terminate_on_pr_merge <> NEW.terminate_on_pr_merge\n    OR OLD.is_pinned <> NEW.is_pinned\n    OR OLD.pinned_at <> NEW.pinned_at\n    OR (OLD.pinned_at IS NULL AND NEW.pinned_at IS NOT NULL)\n    OR (OLD.pinned_at IS NOT NULL AND NEW.pinned_at IS NULL)\n    OR OLD.session_mode <> NEW.session_mode\n    OR OLD.auto_inject_review <> NEW.auto_inject_review\n    OR OLD.auto_review_enabled <> NEW.auto_review_enabled\n    OR OLD.harness <> NEW.harness\n    OR OLD.runtime_launch_id <> NEW.runtime_launch_id\n    OR OLD.agent_session_id <> NEW.agent_session_id\n    OR OLD.native_transcript_path <> NEW.native_transcript_path\n    OR OLD.auto_inject_ci <> NEW.auto_inject_ci\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (NEW.project_id, NEW.id, 'session_updated',\n        json_object(\n            'id', NEW.id,\n            'activity', NEW.activity_state,\n            'isTerminated', json(CASE WHEN NEW.is_terminated THEN 'true' ELSE 'false' END),\n            'terminateOnPrMerge', json(CASE WHEN NEW.terminate_on_pr_merge THEN 'true' ELSE 'false' END),\n            'previewUrl', NEW.preview_url,\n            'previewRevision', NEW.preview_revision,\n            'isPinned', json(CASE WHEN NEW.is_pinned THEN 'true' ELSE 'false' END),\n            'mode', NEW.session_mode,\n            'autoInjectReview', json(CASE WHEN NEW.auto_inject_review THEN 'true' ELSE 'false' END),\n            'autoInjectCI', json(CASE WHEN NEW.auto_inject_ci THEN 'true' ELSE 'false' END),\n            'autoReviewEnabled', json(CASE WHEN NEW.auto_review_enabled THEN 'true' ELSE 'false' END)\n        ),\n        NEW.updated_at);\nEND;",
	},
	{
		name:  "usage_bindings_cdc_insert",
		table: "usage_bindings",
		sql:   "CREATE TRIGGER usage_bindings_cdc_insert AFTER INSERT ON usage_bindings BEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES ((SELECT project_id FROM sessions WHERE id = NEW.session_id),\n            NEW.session_id, 'session_updated', json_object('id', NEW.session_id), NEW.updated_at);\nEND;",
	},
	{
		name:  "usage_bindings_cdc_update",
		table: "usage_bindings",
		sql:   "CREATE TRIGGER usage_bindings_cdc_update AFTER UPDATE ON usage_bindings BEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES ((SELECT project_id FROM sessions WHERE id = NEW.session_id),\n            NEW.session_id, 'session_updated', json_object('id', NEW.session_id), NEW.updated_at);\nEND;",
	},
	{
		name:  "usage_sources_cdc_update",
		table: "usage_sources",
		sql:   "CREATE TRIGGER usage_sources_cdc_update AFTER UPDATE ON usage_sources\nWHEN OLD.anomaly_count IS NOT NEW.anomaly_count\n  OR OLD.last_error_code IS NOT NEW.last_error_code\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    SELECT s.project_id, ub.session_id, 'session_updated', json_object('id', ub.session_id), NEW.updated_at\n    FROM usage_bindings ub JOIN sessions s ON s.id = ub.session_id WHERE ub.id = NEW.binding_id;\nEND;",
	},
	// Outcome contract writers introduced by 0099. Migration 0100 also
	// rebuilds change_log and detaches these; like every writer above they are
	// restored here — never inside migration SQL — because skipped-migration
	// profiles may not have the subject tables when a later rebuild runs.
	{
		name:  "responsibility_outcomes_cdc_insert",
		table: "outcomes",
		deps:  []string{"responsibility_spaces"},
		sql:   "CREATE TRIGGER responsibility_outcomes_cdc_insert\nAFTER INSERT ON outcomes\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT project_id FROM responsibility_spaces WHERE id = NEW.space_id),\n        NULL,\n        'outcome_created',\n        json_object(\n            'id', NEW.id,\n            'spaceId', NEW.space_id,\n            'title', NEW.title,\n            'currentRevisionNumber', NEW.current_revision_number\n        ),\n        NEW.created_at);\nEND;",
	},
	{
		name:  "responsibility_outcomes_cdc_update",
		table: "outcomes",
		deps:  []string{"responsibility_spaces"},
		sql:   "CREATE TRIGGER responsibility_outcomes_cdc_update\nAFTER UPDATE ON outcomes\nWHEN OLD.title <> NEW.title\n     OR OLD.current_revision_number <> NEW.current_revision_number\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT project_id FROM responsibility_spaces WHERE id = NEW.space_id),\n        NULL,\n        'outcome_updated',\n        json_object(\n            'id', NEW.id,\n            'spaceId', NEW.space_id,\n            'title', NEW.title,\n            'previousRevisionNumber', OLD.current_revision_number,\n            'currentRevisionNumber', NEW.current_revision_number\n        ),\n        NEW.updated_at);\nEND;",
	},
	{
		name:  "responsibility_contract_revisions_cdc_insert",
		table: "contract_revisions",
		deps:  []string{"outcomes", "responsibility_spaces"},
		sql:   "CREATE TRIGGER responsibility_contract_revisions_cdc_insert\nAFTER INSERT ON contract_revisions\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT s.project_id\n           FROM outcomes o\n           JOIN responsibility_spaces s ON s.id = o.space_id\n          WHERE o.id = NEW.outcome_id),\n        NULL,\n        'outcome_contract_revised',\n        json_object(\n            'revisionId', NEW.id,\n            'outcomeId', NEW.outcome_id,\n            'number', NEW.number,\n            'goal', NEW.goal\n        ),\n        NEW.created_at);\nEND;",
	},
	// Plan authority writers introduced by 0100.
	{
		name:  "outcome_plans_cdc_insert",
		table: "plan_revisions",
		deps:  []string{"outcomes", "responsibility_spaces"},
		sql:   "CREATE TRIGGER outcome_plans_cdc_insert\nAFTER INSERT ON plan_revisions\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT s.project_id\n           FROM outcomes o\n           JOIN responsibility_spaces s ON s.id = o.space_id\n          WHERE o.id = NEW.outcome_id),\n        NULL,\n        'outcome_plan_proposed',\n        json_object(\n            'planId', NEW.id,\n            'outcomeId', NEW.outcome_id,\n            'number', NEW.number,\n            'contractRevisionNumber', NEW.contract_revision_number,\n            'runBriefCoreDigest', NEW.run_brief_core_digest\n        ),\n        NEW.created_at);\nEND;",
	},
	{
		name:  "outcome_plans_cdc_update",
		table: "plan_revisions",
		deps:  []string{"outcomes", "responsibility_spaces"},
		sql:   "CREATE TRIGGER outcome_plans_cdc_update\nAFTER UPDATE ON plan_revisions\nWHEN OLD.status <> NEW.status\nBEGIN\n    INSERT INTO change_log (project_id, session_id, event_type, payload, created_at)\n    VALUES (\n        (SELECT s.project_id\n           FROM outcomes o\n           JOIN responsibility_spaces s ON s.id = o.space_id\n          WHERE o.id = NEW.outcome_id),\n        NULL,\n        'outcome_plan_approved',\n        json_object(\n            'planId', NEW.id,\n            'outcomeId', NEW.outcome_id,\n            'number', NEW.number,\n            'previousStatus', OLD.status,\n            'status', NEW.status\n        ),\n        datetime('now'));\nEND;",
	},
}

// restoreChangeLogWriters recreates any missing change_log-writing trigger
// whose subject table physically exists. It is idempotent and safe to run on
// every daemon start.
func restoreChangeLogWriters(db *sql.DB) error {
	for _, w := range changeLogWriters {
		var n int
		if err := db.QueryRow(
			"SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name=?", w.name,
		).Scan(&n); err != nil {
			return fmt.Errorf("inspect trigger %s: %w", w.name, err)
		}
		if n > 0 {
			continue
		}
		var t int
		if err := db.QueryRow(
			"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", w.table,
		).Scan(&t); err != nil {
			return fmt.Errorf("inspect table %s: %w", w.table, err)
		}
		if t == 0 {
			// Degraded profile: the subject table arrives through a later
			// schema repair together with its writer.
			continue
		}
		ready := true
		for _, dep := range w.deps {
			var d int
			if err := db.QueryRow(
				"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", dep,
			).Scan(&d); err != nil {
				return fmt.Errorf("inspect dependency %s of %s: %w", dep, w.name, err)
			}
			if d == 0 {
				ready = false
				break
			}
		}
		if !ready {
			// A trigger body that joins an absent table would abort every
			// future write to its subject; defer until dependencies land.
			continue
		}
		if _, err := db.Exec(w.sql); err != nil {
			return fmt.Errorf("restore trigger %s: %w", w.name, err)
		}
	}
	return nil
}
