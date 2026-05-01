-- +goose Up
CREATE TABLE bands (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    timezone TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE users (
    id UUID PRIMARY KEY,
    auth_provider TEXT NOT NULL,
    auth_provider_user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (auth_provider, auth_provider_user_id),
    UNIQUE (email)
);

CREATE TABLE band_memberships (
    id UUID PRIMARY KEY,
    band_id UUID NOT NULL REFERENCES bands (id),
    user_id UUID NOT NULL REFERENCES users (id),
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (band_id, user_id)
);

CREATE TABLE band_invites (
    id UUID PRIMARY KEY,
    band_id UUID NOT NULL REFERENCES bands (id),
    email TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('viewer')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'revoked', 'expired')),
    invited_by_user_id UUID NOT NULL REFERENCES users (id),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id),
    band_id UUID NOT NULL REFERENCES bands (id),
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    request_id TEXT NOT NULL,
    idempotency_key TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE idempotency_records (
    id UUID PRIMARY KEY,
    band_id UUID NOT NULL REFERENCES bands (id),
    operation TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response_body JSONB NOT NULL,
    status_code INTEGER NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (band_id, operation, idempotency_key)
);

CREATE INDEX audit_logs_band_created_at_idx ON audit_logs (band_id, created_at);
CREATE UNIQUE INDEX band_invites_pending_email_idx ON band_invites (band_id, email) WHERE status = 'pending';
CREATE INDEX idempotency_records_expires_at_idx ON idempotency_records (expires_at);

-- +goose Down
DROP TABLE idempotency_records;
DROP TABLE audit_logs;
DROP TABLE band_invites;
DROP TABLE band_memberships;
DROP TABLE users;
DROP TABLE bands;
