-- Scoped exception for explicit Outcome erasure; normal deletes remain guarded.
DROP TRIGGER IF EXISTS acceptance_decisions_immutable_delete;
CREATE TRIGGER acceptance_decisions_immutable_delete BEFORE DELETE ON acceptance_decisions WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='acceptance_decisions' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'acceptance decisions are append-only'); END;
DROP TRIGGER IF EXISTS attempt_artifact_files_frozen_delete;
CREATE TRIGGER attempt_artifact_files_frozen_delete
BEFORE DELETE ON attempt_artifact_files WHEN ((SELECT frozen_at FROM attempt_receipts WHERE attempt_id = OLD.attempt_id) IS NOT NULL) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='attempt_artifact_files' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'a frozen attempt receipt cannot be replaced');
END;
DROP TRIGGER IF EXISTS attempt_check_runs_no_delete;
CREATE TRIGGER attempt_check_runs_no_delete
BEFORE DELETE ON attempt_check_runs WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='attempt_check_runs' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'recorded check observations are immutable');
END;
DROP TRIGGER IF EXISTS attempt_fences_immutable_delete;
CREATE TRIGGER attempt_fences_immutable_delete
BEFORE DELETE ON attempt_fences WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='attempt_fences' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'attempt fences are append-only');
END;
DROP TRIGGER IF EXISTS attempt_observations_immutable_delete;
CREATE TRIGGER attempt_observations_immutable_delete
BEFORE DELETE ON attempt_observations WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='attempt_observations' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'attempt observations are append-only');
END;
DROP TRIGGER IF EXISTS attempt_receipts_frozen_delete;
CREATE TRIGGER attempt_receipts_frozen_delete
BEFORE DELETE ON attempt_receipts WHEN (OLD.frozen_at IS NOT NULL) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='attempt_receipts' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'a frozen attempt receipt cannot be deleted');
END;
DROP TRIGGER IF EXISTS attempt_recovery_receipts_immutable_delete;
CREATE TRIGGER attempt_recovery_receipts_immutable_delete
BEFORE DELETE ON attempt_recovery_receipts WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='attempt_recovery_receipts' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'recovery receipts are append-only');
END;
DROP TRIGGER IF EXISTS attempt_sessions_immutable_delete;
CREATE TRIGGER attempt_sessions_immutable_delete
BEFORE DELETE ON attempt_sessions WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='attempt_sessions' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'attempt session refs are immutable');
END;
DROP TRIGGER IF EXISTS attempts_immutable_delete;
CREATE TRIGGER attempts_immutable_delete
BEFORE DELETE ON attempts WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='attempts' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'attempts are append-only');
END;
DROP TRIGGER IF EXISTS capability_grants_immutable_delete;
CREATE TRIGGER capability_grants_immutable_delete
BEFORE DELETE ON capability_grants WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='capability_grants' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'capability grants are immutable');
END;
DROP TRIGGER IF EXISTS contract_criteria_immutable_delete;
CREATE TRIGGER contract_criteria_immutable_delete BEFORE DELETE ON contract_criteria WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='contract_criteria' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'contract criteria are append-only'); END;
DROP TRIGGER IF EXISTS contract_revision_intake_core_immutable_delete;
CREATE TRIGGER contract_revision_intake_core_immutable_delete BEFORE DELETE ON contract_revision_intake_core WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='contract_revision_intake_core' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'contract intake core is append-only'); END;
DROP TRIGGER IF EXISTS contract_revisions_immutable_delete;
CREATE TRIGGER contract_revisions_immutable_delete
BEFORE DELETE ON contract_revisions WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='contract_revisions' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'contract revisions are immutable');
END;
DROP TRIGGER IF EXISTS contribution_dependencies_immutable_delete;
CREATE TRIGGER contribution_dependencies_immutable_delete
BEFORE DELETE ON contribution_dependencies WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='contribution_dependencies' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'contribution dependencies are append-only'); END;
DROP TRIGGER IF EXISTS contribution_links_immutable_delete;
CREATE TRIGGER contribution_links_immutable_delete
BEFORE DELETE ON contribution_links WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='contribution_links' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'contribution links are append-only'); END;
DROP TRIGGER IF EXISTS contribution_waivers_immutable_delete;
CREATE TRIGGER contribution_waivers_immutable_delete
BEFORE DELETE ON contribution_dependency_waivers WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='contribution_dependency_waivers' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'dependency waivers are append-only'); END;
DROP TRIGGER IF EXISTS decomposition_contributions_immutable_delete;
CREATE TRIGGER decomposition_contributions_immutable_delete
BEFORE DELETE ON decomposition_contributions WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='decomposition_contributions' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'decomposition contributions are append-only'); END;
DROP TRIGGER IF EXISTS decomposition_retained_immutable_delete;
CREATE TRIGGER decomposition_retained_immutable_delete
BEFORE DELETE ON decomposition_retained_criteria WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='decomposition_retained_criteria' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'retained criteria are append-only'); END;
DROP TRIGGER IF EXISTS decomposition_revisions_immutable_delete;
CREATE TRIGGER decomposition_revisions_immutable_delete
BEFORE DELETE ON decomposition_revisions WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='decomposition_revisions' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'decomposition revisions are append-only'); END;
DROP TRIGGER IF EXISTS evidence_items_immutable_delete;
CREATE TRIGGER evidence_items_immutable_delete BEFORE DELETE ON evidence_items WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='evidence_items' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'evidence items are append-only'); END;
DROP TRIGGER IF EXISTS intake_clarification_answers_immutable_delete;
CREATE TRIGGER intake_clarification_answers_immutable_delete BEFORE DELETE ON intake_clarification_answers WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='intake_clarification_answers' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'intake clarification answers are append-only'); END;
DROP TRIGGER IF EXISTS intake_clarifications_immutable_delete;
CREATE TRIGGER intake_clarifications_immutable_delete BEFORE DELETE ON intake_clarifications WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='intake_clarifications' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'intake clarifications are append-only'); END;
DROP TRIGGER IF EXISTS intake_confirmations_immutable_delete;
CREATE TRIGGER intake_confirmations_immutable_delete BEFORE DELETE ON intake_confirmations WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='intake_confirmations' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'intake confirmations are append-only'); END;
DROP TRIGGER IF EXISTS intake_conversation_refs_immutable_delete;
CREATE TRIGGER intake_conversation_refs_immutable_delete BEFORE DELETE ON intake_conversation_refs WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='intake_conversation_refs' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'intake conversation refs are append-only'); END;
DROP TRIGGER IF EXISTS intake_proposals_immutable_delete;
CREATE TRIGGER intake_proposals_immutable_delete BEFORE DELETE ON intake_proposal_revisions WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='intake_proposal_revisions' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'intake proposals are append-only'); END;
DROP TRIGGER IF EXISTS outcome_corrections_immutable_delete;
CREATE TRIGGER outcome_corrections_immutable_delete BEFORE DELETE ON outcome_corrections WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='outcome_corrections' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'outcome corrections are append-only'); END;
DROP TRIGGER IF EXISTS outcome_deliveries_immutable_delete;
CREATE TRIGGER outcome_deliveries_immutable_delete
BEFORE DELETE ON outcome_deliveries WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='outcome_deliveries' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'delivery records are immutable');
END;
DROP TRIGGER IF EXISTS outcome_document_contexts_immutable_delete;
CREATE TRIGGER outcome_document_contexts_immutable_delete
BEFORE DELETE ON outcome_document_contexts WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='outcome_document_contexts' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'document context revisions are immutable');
END;
DROP TRIGGER IF EXISTS outcome_document_sources_immutable_delete;
CREATE TRIGGER outcome_document_sources_immutable_delete
BEFORE DELETE ON outcome_document_sources WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='outcome_document_sources' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'document context revisions are immutable');
END;
DROP TRIGGER IF EXISTS outcome_run_intents_immutable_delete;
CREATE TRIGGER outcome_run_intents_immutable_delete
BEFORE DELETE ON outcome_run_intents WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='outcome_run_intents' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'run intent generations are immutable');
END;
DROP TRIGGER IF EXISTS plan_revisions_immutable_delete;
CREATE TRIGGER plan_revisions_immutable_delete
BEFORE DELETE ON plan_revisions WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='plan_revisions' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'plan revisions are immutable');
END;
DROP TRIGGER IF EXISTS planning_turns_immutable_delete;
CREATE TRIGGER planning_turns_immutable_delete
BEFORE DELETE ON planning_turns WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='planning_turns' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'planning turns are immutable');
END;
DROP TRIGGER IF EXISTS responsibility_links_immutable_delete;
CREATE TRIGGER responsibility_links_immutable_delete BEFORE DELETE ON responsibility_links WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='responsibility_links' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'responsibility links are append-only'); END;
DROP TRIGGER IF EXISTS verification_runs_immutable_delete;
CREATE TRIGGER verification_runs_immutable_delete BEFORE DELETE ON verification_runs WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='verification_runs' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'verification runs are append-only'); END;
DROP TRIGGER IF EXISTS work_unit_checks_immutable_delete;
CREATE TRIGGER work_unit_checks_immutable_delete
BEFORE DELETE ON work_unit_checks WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='work_unit_checks' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'approved checks are immutable');
END;
DROP TRIGGER IF EXISTS work_unit_criterion_bindings_immutable_delete;
CREATE TRIGGER work_unit_criterion_bindings_immutable_delete
BEFORE DELETE ON work_unit_criterion_bindings WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='work_unit_criterion_bindings' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'work unit criterion bindings are immutable'); END;
DROP TRIGGER IF EXISTS work_unit_dependencies_immutable_delete;
CREATE TRIGGER work_unit_dependencies_immutable_delete
BEFORE DELETE ON work_unit_dependencies WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='work_unit_dependencies' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'work unit dependencies are immutable'); END;
DROP TRIGGER IF EXISTS work_unit_provider_bindings_immutable_delete;
CREATE TRIGGER work_unit_provider_bindings_immutable_delete
BEFORE DELETE ON work_unit_provider_bindings WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='work_unit_provider_bindings' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'work unit provider bindings are immutable');
END;
DROP TRIGGER IF EXISTS work_unit_required_capabilities_immutable_delete;
CREATE TRIGGER work_unit_required_capabilities_immutable_delete
BEFORE DELETE ON work_unit_required_capabilities WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='work_unit_required_capabilities' AND row_id=OLD.rowid)
BEGIN SELECT RAISE(ABORT, 'work unit required capabilities are immutable'); END;
DROP TRIGGER IF EXISTS work_units_immutable_delete;
CREATE TRIGGER work_units_immutable_delete
BEFORE DELETE ON work_units WHEN (1) AND NOT EXISTS (SELECT 1 FROM outcome_purge_scope WHERE table_name='work_units' AND row_id=OLD.rowid)
BEGIN
    SELECT RAISE(ABORT, 'work units are immutable');
END;
