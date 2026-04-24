-- +goose Up
-- Base schema: extensions and common trigger functions.
-- Register db.BaseSchema (priority 0) before feature schemas.

CREATE EXTENSION IF NOT EXISTS citext;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- REVIEW: auto-generated Down SQL — verify before committing
DROP FUNCTION IF EXISTS update_updated_at();
