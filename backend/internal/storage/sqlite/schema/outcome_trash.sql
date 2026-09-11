
DROP TRIGGER IF EXISTS outcome_trash_attempt_guard;
CREATE TRIGGER outcome_trash_attempt_guard BEFORE INSERT ON attempts
WHEN EXISTS (SELECT 1 FROM outcome_trash WHERE outcome_id=NEW.outcome_id)
BEGIN SELECT RAISE(ABORT, 'Outcome is in Trash'); END;
DROP TRIGGER IF EXISTS outcome_trash_plan_guard;
CREATE TRIGGER outcome_trash_plan_guard BEFORE INSERT ON plan_revisions
WHEN EXISTS (SELECT 1 FROM outcome_trash WHERE outcome_id=NEW.outcome_id)
BEGIN SELECT RAISE(ABORT, 'Outcome is in Trash'); END;
DROP TRIGGER IF EXISTS outcome_trash_contract_guard;
CREATE TRIGGER outcome_trash_contract_guard BEFORE INSERT ON contract_revisions
WHEN EXISTS (SELECT 1 FROM outcome_trash WHERE outcome_id=NEW.outcome_id)
BEGIN SELECT RAISE(ABORT, 'Outcome is in Trash'); END;
DROP TRIGGER IF EXISTS outcome_trash_run_guard;
CREATE TRIGGER outcome_trash_run_guard BEFORE INSERT ON outcome_run_intents
WHEN EXISTS (SELECT 1 FROM outcome_trash WHERE outcome_id=NEW.outcome_id)
BEGIN SELECT RAISE(ABORT, 'Outcome is in Trash'); END;
DROP TRIGGER IF EXISTS outcome_trash_child_guard;
CREATE TRIGGER outcome_trash_child_guard BEFORE INSERT ON outcomes
WHEN EXISTS (SELECT 1 FROM outcome_trash WHERE outcome_id=NEW.parent_outcome_id)
BEGIN SELECT RAISE(ABORT, 'Parent Outcome is in Trash'); END;
DROP TRIGGER IF EXISTS outcome_trash_session_resume_guard;
CREATE TRIGGER outcome_trash_session_resume_guard BEFORE UPDATE OF is_terminated ON sessions
WHEN NEW.is_terminated=0 AND EXISTS (
 SELECT 1 FROM attempt_sessions a JOIN attempts t ON t.id=a.attempt_id
 JOIN outcome_trash x ON x.outcome_id=t.outcome_id WHERE a.session_id=NEW.id
)
BEGIN SELECT RAISE(ABORT, 'Outcome is in Trash'); END;
DROP TRIGGER IF EXISTS outcome_trash_delivery_guard;
CREATE TRIGGER outcome_trash_delivery_guard BEFORE INSERT ON outcome_deliveries
WHEN EXISTS (SELECT 1 FROM outcome_trash WHERE outcome_id=NEW.outcome_id)
BEGIN SELECT RAISE(ABORT, 'Outcome is in Trash'); END;
DROP TRIGGER IF EXISTS outcome_trash_document_guard;
CREATE TRIGGER outcome_trash_document_guard BEFORE INSERT ON outcome_document_contexts
WHEN EXISTS (SELECT 1 FROM outcome_trash WHERE outcome_id=NEW.outcome_id)
BEGIN SELECT RAISE(ABORT, 'Outcome is in Trash'); END;
DROP TRIGGER IF EXISTS outcome_delete_cdc;
CREATE TRIGGER outcome_delete_cdc AFTER DELETE ON outcomes
BEGIN
 INSERT INTO change_log(project_id,session_id,event_type,payload,created_at)
 VALUES((SELECT project_id FROM responsibility_spaces WHERE id=OLD.space_id),NULL,'outcome_updated',json_object('outcomeId',OLD.id,'deleted',1),datetime('now'));
END;
