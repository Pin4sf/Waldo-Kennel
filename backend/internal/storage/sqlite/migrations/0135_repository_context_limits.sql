-- Owner-configurable bounds on the daemon's bounded repository-context
-- packet, used for Waldo reasoning (intake analysis, planning). These were
-- previously fixed Go constants; a real repository during the launch
-- stabilization end-to-end exercise showed the fixed discovery-walk bound was
-- too low for an ordinary repository, and the caller that consumed a partial
-- snapshot refused ALL planning outright rather than proceeding with a
-- disclosed partial context. Each column is nullable: NULL keeps Kennel's
-- built-in default (unchanged behavior for every existing profile); zero
-- means uncapped.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_settings ADD COLUMN repository_context_max_files INTEGER;
ALTER TABLE app_settings ADD COLUMN repository_context_max_bytes INTEGER;
ALTER TABLE app_settings ADD COLUMN repository_context_max_visited INTEGER;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_settings DROP COLUMN repository_context_max_visited;
ALTER TABLE app_settings DROP COLUMN repository_context_max_bytes;
ALTER TABLE app_settings DROP COLUMN repository_context_max_files;
-- +goose StatementEnd
