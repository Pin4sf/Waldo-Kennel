-- Project-scoped Chat conversations are the durable narrative for Outcome
-- orchestrators. Project their latest settled assistant message onto the bound
-- session so the session read model can derive Outcome/Kanban lifecycle without
-- scraping a live provider process.

-- +goose Up
-- +goose StatementBegin
CREATE TRIGGER conversation_assistant_insert_session_projection
AFTER INSERT ON conversation_messages
WHEN NEW.role = 'assistant' AND NEW.streaming = 0
BEGIN
    UPDATE sessions
    SET latest_assistant_update = NEW.text,
        updated_at = CASE WHEN updated_at < NEW.updated_at THEN NEW.updated_at ELSE updated_at END
    WHERE id = (
        SELECT current_session_id
        FROM conversations
        WHERE id = NEW.conversation_id
    )
      AND session_mode = 'chat'
      AND is_terminated = 0;
END;

CREATE TRIGGER conversation_assistant_settle_session_projection
AFTER UPDATE OF text, streaming ON conversation_messages
WHEN NEW.role = 'assistant'
 AND NEW.streaming = 0
 AND (OLD.streaming <> 0 OR OLD.text <> NEW.text)
BEGIN
    UPDATE sessions
    SET latest_assistant_update = NEW.text,
        updated_at = CASE WHEN updated_at < NEW.updated_at THEN NEW.updated_at ELSE updated_at END
    WHERE id = (
        SELECT current_session_id
        FROM conversations
        WHERE id = NEW.conversation_id
    )
      AND session_mode = 'chat'
      AND is_terminated = 0;
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS conversation_assistant_settle_session_projection;
DROP TRIGGER IF EXISTS conversation_assistant_insert_session_projection;
-- +goose StatementEnd
