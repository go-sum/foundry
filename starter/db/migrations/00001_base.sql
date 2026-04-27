-- Auto-generated migration - do not edit

-- +migrate Up
CREATE EXTENSION IF NOT EXISTS citext;

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- +migrate Down
DROP EXTENSION IF EXISTS citext;

DROP FUNCTION IF EXISTS update_updated_at;
