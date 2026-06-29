CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    name text NOT NULL,
    email text UNIQUE NOT NULL,
    password_hash text,
    oauth_provider text,
    oauth_id text,
    role TEXT NOT NULL DEFAULT 'student' CHECK (ROLE IN ('student', 'instructor')),
    userpfpUrl text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    CHECK ((password_hash IS NOT NULL AND oauth_provider IS NULL AND oauth_id IS NULL) OR (password_hash IS NULL AND oauth_provider IS NOT NULL AND oauth_id IS NOT NULL))
);

CREATE UNIQUE INDEX users_email_lower_idx ON users (lower(email));

CREATE TABLE IF NOT EXISTS refresh_tokens (
    token_id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked boolean NOT NULL DEFAULT FALSE,
    replaced_by uuid,
    user_agent text,
    ip_address text
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    user_id uuid UNIQUE NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token text NOT NULL,
    expires_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS courses (
    course_id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    creator_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title text NOT NULL,
    slug text UNIQUE NOT NULL,
    description text,
    price int NOT NULL,
    currency text NOT NULL,
    published boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payments (
    payment_id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    course_id uuid NOT NULL REFERENCES courses (course_id) ON DELETE CASCADE,
    provider text NOT NULL,
    provider_id text NOT NULL,
    amount int NOT NULL,
    currency text NOT NULL,
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_user ON payments (user_id);

CREATE INDEX idx_payments_course ON payments (course_id);

CREATE TABLE IF NOT EXISTS course_enrollments (
    enrollment_id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    course_id uuid NOT NULL REFERENCES courses (course_id) ON DELETE CASCADE,
    enrolled_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    payment_id uuid REFERENCES payments (payment_id) ON DELETE SET NULL,
    UNIQUE (user_id, course_id)
);

CREATE INDEX idx_course_enrollments_user ON course_enrollments (user_id);

CREATE INDEX idx_course_enrollments_course ON course_enrollments (course_id);

CREATE TABLE sections (
    section_id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    course_id uuid NOT NULL REFERENCES courses (course_id) ON DELETE CASCADE,
    title text NOT NULL,
    position int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sections_course ON sections (course_id);

CREATE TABLE lessons (
    lesson_id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    section_id uuid NOT NULL REFERENCES sections (section_id) ON DELETE CASCADE,
    title text NOT NULL,
    content text,
    position int NOT NULL
);

CREATE INDEX idx_lessons_section ON lessons (section_id);

CREATE TABLE IF NOT EXISTS progress (
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    lesson_id uuid NOT NULL REFERENCES lessons (lesson_id) ON DELETE CASCADE,
    completed_at timestamptz,
    PRIMARY KEY (user_id, lesson_id)
);

CREATE INDEX idx_progress_user ON progress (user_id);

CREATE INDEX idx_progress_lesson ON progress (lesson_id);

CREATE TABLE IF NOT EXISTS video_uploads (
    upload_id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    lesson_id uuid NOT NULL REFERENCES lessons (lesson_id) ON DELETE CASCADE,
    s3_key text NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'ready', 'error')),
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_video_uploads_lesson ON video_uploads (lesson_id);

