-- +goose Up
ALTER TABLE band_invites
    ADD COLUMN token_hash TEXT,
    ADD COLUMN accepted_by_user_id UUID REFERENCES users (id),
    ADD COLUMN accepted_at TIMESTAMPTZ,
    ADD COLUMN revoked_at TIMESTAMPTZ;

CREATE UNIQUE INDEX band_invites_token_hash_idx
    ON band_invites (token_hash)
    WHERE token_hash IS NOT NULL;

CREATE INDEX band_invites_band_status_created_at_idx
    ON band_invites (band_id, status, created_at DESC);

-- +goose Down
DROP INDEX band_invites_band_status_created_at_idx;
DROP INDEX band_invites_token_hash_idx;

ALTER TABLE band_invites
    DROP COLUMN revoked_at,
    DROP COLUMN accepted_at,
    DROP COLUMN accepted_by_user_id,
    DROP COLUMN token_hash;
