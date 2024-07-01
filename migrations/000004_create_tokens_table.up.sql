CREATE TABLE IF NOT EXISTS tokens(
    hash bytea PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    expiry_time timestamp(0) WITH TIME ZONE NOT NULL,
    scope TEXT NOT NULL
);