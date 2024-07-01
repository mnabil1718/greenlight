CREATE TABLE IF NOT EXISTS users(
    id bigserial PRIMARY KEY,
    name text NOT NULL,
    email citext UNIQUE NOT NULL,
    password bytea NOT NULL,
    activated bool NOT NULL DEFAULT false,
    version integer NOT NULL DEFAULT 1,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);