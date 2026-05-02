-- +goose Up
CREATE TABLE calendar_events (
    id UUID PRIMARY KEY,
    band_id UUID NOT NULL REFERENCES bands (id),
    created_by_user_id UUID NOT NULL REFERENCES users (id),
    event_type TEXT NOT NULL CHECK (event_type IN ('show', 'rehearsal', 'release', 'meeting', 'task', 'other')),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    location_name TEXT NOT NULL,
    address TEXT NOT NULL,
    starts_at_local TIMESTAMP NOT NULL,
    ends_at_local TIMESTAMP NOT NULL,
    timezone TEXT NOT NULL,
    recurrence_frequency TEXT NOT NULL CHECK (recurrence_frequency IN ('none', 'daily', 'weekly', 'monthly')),
    recurrence_interval INTEGER NOT NULL CHECK (recurrence_interval >= 0),
    recurrence_ends_on DATE,
    recurrence_count INTEGER CHECK (recurrence_count IS NULL OR recurrence_count > 0),
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users (id),
    idempotency_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK (ends_at_local > starts_at_local),
    CHECK (
        (recurrence_frequency = 'none' AND recurrence_interval = 0 AND recurrence_ends_on IS NULL AND recurrence_count IS NULL)
        OR
        (recurrence_frequency <> 'none' AND recurrence_interval > 0)
    ),
    CHECK (recurrence_ends_on IS NULL OR recurrence_count IS NULL)
);

CREATE INDEX calendar_events_band_active_starts_idx
    ON calendar_events (band_id, starts_at_local)
    WHERE deleted_at IS NULL;

CREATE INDEX calendar_events_band_active_ends_idx
    ON calendar_events (band_id, ends_at_local)
    WHERE deleted_at IS NULL;

CREATE INDEX calendar_events_band_active_recurrence_idx
    ON calendar_events (band_id, recurrence_frequency, recurrence_ends_on)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX calendar_events_band_active_recurrence_idx;
DROP INDEX calendar_events_band_active_ends_idx;
DROP INDEX calendar_events_band_active_starts_idx;
DROP TABLE calendar_events;
