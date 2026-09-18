CREATE TABLE users (
 id text PRIMARY KEY, email text UNIQUE NOT NULL, password_hash text NOT NULL,
 timezone text NOT NULL DEFAULT 'Asia/Bangkok', created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE sessions (
 token_hash text PRIMARY KEY, user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 csrf text NOT NULL, expires_at timestamptz NOT NULL
);
CREATE TABLE projects (
 id text PRIMARY KEY, user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 name text NOT NULL, color text NOT NULL DEFAULT '#537868', UNIQUE(user_id,name)
);
CREATE TABLE notes (
 id text PRIMARY KEY, user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 work_date date NOT NULL, title text NOT NULL, project text NOT NULL DEFAULT '',
 description text NOT NULL DEFAULT '', tasks jsonb NOT NULL DEFAULT '[]',
 status text NOT NULL CHECK (status IN ('todo','progress','completed','blocked')),
 priority text NOT NULL CHECK (priority IN ('low','medium','high')),
 tags text[] NOT NULL DEFAULT '{}', minutes integer NOT NULL CHECK(minutes BETWEEN 0 AND 1440),
 blockers text NOT NULL DEFAULT '', next_steps text NOT NULL DEFAULT '',
 version integer NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notes_owner_date ON notes(user_id,work_date DESC,id);
CREATE TABLE attachments (
 id text PRIMARY KEY, note_id text NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
 user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 name text NOT NULL, mime text NOT NULL, size bigint NOT NULL CHECK(size > 0),
 storage_key text UNIQUE NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE file_deletions (storage_key text PRIMARY KEY, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE email_attempts (
 id text PRIMARY KEY, user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 idempotency_key text NOT NULL, fingerprint text NOT NULL,
 recipients jsonb NOT NULL, subject text NOT NULL,
 outcome text NOT NULL CHECK(outcome IN ('sending','accepted','failed','unknown')),
 detail text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(user_id,idempotency_key)
);
