-- +goose Up

CREATE TABLE idempotency_keys (
    key          UUID PRIMARY KEY,
    request_hash BYTEA NOT NULL,
    -- NULL exists only while the surrounding creation transaction is open;
    -- it is filled before commit.
    trip_id      UUID REFERENCES trips(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idempotency_keys_created_at_idx ON idempotency_keys (created_at);

-- +goose Down

DROP INDEX IF EXISTS idempotency_keys_created_at_idx;
DROP TABLE IF EXISTS idempotency_keys;
