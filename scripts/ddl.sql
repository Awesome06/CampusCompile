CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ==========================================
-- 1. CUSTOM TYPES
-- ==========================================
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('student', 'professor', 'admin');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'problem_difficulty') THEN
        CREATE TYPE problem_difficulty AS ENUM ('Easy', 'Medium', 'Hard');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'submission_status') THEN
        CREATE TYPE submission_status AS ENUM ('Pending', 'Running', 'AC', 'WA', 'TLE', 'MLE', 'RE', 'CE', 'SE');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'telemetry_event_type') THEN
        CREATE TYPE telemetry_event_type AS ENUM ('blur', 'paste_attempt', 'autotyper_suspected', 'visibility_spoof_suspected', 'anomalous_routing', 'fullscreen_dropped');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'audit_status') THEN
        CREATE TYPE audit_status AS ENUM ('pending', 'in_progress', 'completed', 'failed');
    END IF;
END $$;

-- ==========================================
-- 2. CORE ENTITIES
-- ==========================================

-- Users table: Stores all user profiles and authentication mappings
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id VARCHAR(255) UNIQUE,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE,
    real_name VARCHAR(255),
    batch VARCHAR(50),
    section VARCHAR(50),
    student_group VARCHAR(50),
    course VARCHAR(100),
    department VARCHAR(100),
    graduation_year INTEGER,
    role user_role NOT NULL DEFAULT 'student',
    campus_rating INTEGER DEFAULT 1200,
    is_onboarded BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Problems table: Stores the problem statements, limits, and checker metadata
CREATE TABLE IF NOT EXISTS problems (
    problem_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT NOT NULL,
    difficulty problem_difficulty NOT NULL,
    time_limit_ms INTEGER NOT NULL DEFAULT 2000,
    memory_limit_kb INTEGER NOT NULL DEFAULT 262144,
    author_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    has_checker BOOLEAN DEFAULT false,
    checker_s3_key TEXT,
    is_public BOOLEAN DEFAULT false,
    fts tsvector GENERATED ALWAYS AS (
        to_tsvector('english', title)
    ) STORED
);

-- Contests table: Manages competition windows and rules
CREATE TABLE IF NOT EXISTS contests (
    contest_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    host_organization VARCHAR(255),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    access_rules JSONB,
    author_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
    is_public BOOLEAN NOT NULL DEFAULT false,
    moss_audit_status audit_status DEFAULT 'pending',
    fts tsvector GENERATED ALWAYS AS (
        to_tsvector('english', title)
    ) STORED
);

-- Playlists table: Stores curated lists of problems
CREATE TABLE IF NOT EXISTS playlists (
    playlist_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    author_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
    is_public BOOLEAN NOT NULL DEFAULT false,
    overall_difficulty problem_difficulty NOT NULL DEFAULT 'Easy',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    fts tsvector GENERATED ALWAYS AS (
        to_tsvector('english', title)
    ) STORED
);

-- ==========================================
-- 3. RELATIONAL & ACTIVITY TABLES
-- ==========================================

-- Contest Registrations: Maps users to the contests they joined
CREATE TABLE IF NOT EXISTS contest_registrations (
    contest_id UUID REFERENCES contests(contest_id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    registered_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (contest_id, user_id)
);

-- Contest Problems: Maps specific problems to specific contests
CREATE TABLE IF NOT EXISTS contest_problems (
    contest_id UUID REFERENCES contests(contest_id) ON DELETE CASCADE,
    problem_id UUID REFERENCES problems(problem_id) ON DELETE CASCADE,
    points_value INTEGER NOT NULL DEFAULT 100,
    PRIMARY KEY (contest_id, problem_id)
);

-- Playlist Problems: Maps specific problems to specific playlists
CREATE TABLE IF NOT EXISTS playlist_problems (
    playlist_id UUID REFERENCES playlists(playlist_id) ON DELETE CASCADE,
    problem_id UUID REFERENCES problems(problem_id) ON DELETE CASCADE,
    order_index INTEGER NOT NULL,
    custom_difficulty problem_difficulty,
    PRIMARY KEY (playlist_id, problem_id),
    CONSTRAINT unique_playlist_order UNIQUE (playlist_id, order_index)
);

-- Central Tags Table: Stores all unique problem and playlist tags
CREATE TABLE IF NOT EXISTS tags (
    tag_id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL
);

-- Problem Tags: Maps generic tags directly to the problem
CREATE TABLE IF NOT EXISTS problem_tags (
    problem_id UUID REFERENCES problems(problem_id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (problem_id, tag_id)
);

-- Playlist Tags: Maps generic topic tags directly to the playlist
CREATE TABLE IF NOT EXISTS playlist_tags (
    playlist_id UUID REFERENCES playlists(playlist_id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (playlist_id, tag_id)
);


-- Submissions table: Tracks code execution runs
CREATE TABLE IF NOT EXISTS submissions (
    submission_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    problem_id UUID REFERENCES problems(problem_id) ON DELETE CASCADE,
    contest_id UUID REFERENCES contests(contest_id) ON DELETE CASCADE,
    language VARCHAR(50) NOT NULL,
    status submission_status DEFAULT 'Pending',
    execution_time_ms INTEGER,
    memory_used_kb INTEGER,
    submitted_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    error_logs TEXT,
    source_code_s3_key VARCHAR(512)
);

-- Test Cases: Stores I/O data or S3 references for validating submissions
CREATE TABLE IF NOT EXISTS test_cases (
    test_case_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    problem_id UUID REFERENCES problems(problem_id) ON DELETE CASCADE,
    is_hidden BOOLEAN DEFAULT true,
    input_s3_key VARCHAR(512) NOT NULL,
    expected_s3_key VARCHAR(512) NOT NULL
);

-- ==========================================
-- 4. ANTI-CHEAT & TELEMETRY
-- ==========================================

-- Contest Telemetry: Logs suspicious browser and editor activity
CREATE TABLE IF NOT EXISTS contest_telemetry (
    telemetry_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id UUID REFERENCES contests(contest_id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    event_type telemetry_event_type NOT NULL,
    metadata JSONB, 
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Plagiarism Reports: Stores MOSS audit results from the Python worker
CREATE TABLE IF NOT EXISTS plagiarism_reports (
    report_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id UUID REFERENCES contests(contest_id) ON DELETE CASCADE,
    problem_id UUID REFERENCES problems(problem_id) ON DELETE CASCADE,
    user_1_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    user_2_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    similarity_score NUMERIC(5,2) NOT NULL,
    moss_url TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- ==========================================
-- 5. INDEXES
-- ==========================================

-- Contests
CREATE INDEX IF NOT EXISTS idx_contests_author_id ON contests(author_id);
CREATE INDEX IF NOT EXISTS idx_contests_times ON contests(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_contests_moss_audit ON contests(moss_audit_status, end_time, updated_at);
CREATE INDEX IF NOT EXISTS idx_contests_fts ON contests USING GIN (fts);

-- Playlists
CREATE INDEX IF NOT EXISTS idx_playlists_author_id ON playlists(author_id);
CREATE INDEX IF NOT EXISTS idx_playlists_fts ON playlists USING GIN (fts);

-- Playlist Problems
CREATE INDEX IF NOT EXISTS idx_playlist_problems_playlist_id ON playlist_problems(playlist_id);

-- Problems
CREATE INDEX IF NOT EXISTS idx_problems_fts ON problems USING GIN (fts);
CREATE INDEX IF NOT EXISTS idx_problems_created_at ON problems(created_at DESC);

-- Contest Registrations
CREATE INDEX IF NOT EXISTS idx_contest_registrations_user_id ON contest_registrations(user_id);

-- Submissions
-- --------------------------------------------------------------------------------
-- Phase 3 Indices
-- --------------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_submissions_user_problem_time ON submissions(user_id, problem_id, submitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_playlist_problems_problem_id ON playlist_problems(problem_id);
CREATE INDEX IF NOT EXISTS idx_submissions_contest_id ON submissions(contest_id);
CREATE INDEX IF NOT EXISTS idx_submissions_problem_id ON submissions(problem_id);
CREATE INDEX IF NOT EXISTS idx_submissions_user_id ON submissions(user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
CREATE INDEX IF NOT EXISTS idx_submissions_submitted_at ON submissions(submitted_at DESC);

-- Telemetry & Plagiarism
CREATE INDEX IF NOT EXISTS idx_telemetry_contest_user ON contest_telemetry(contest_id, user_id);
CREATE INDEX IF NOT EXISTS idx_plagiarism_contest_problem ON plagiarism_reports(contest_id, problem_id);


-- ==========================================
-- 6. DATABASE TRIGGERS
-- ==========================================

-- Reusable function to bump the updated_at timestamp
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Attach the auto-update trigger to the contests table
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'contests') THEN
        DROP TRIGGER IF EXISTS update_contests_modtime ON contests;
        CREATE TRIGGER update_contests_modtime
            BEFORE UPDATE ON contests
            FOR EACH ROW
            EXECUTE FUNCTION update_modified_column();
    END IF;

    IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'playlists') THEN
        DROP TRIGGER IF EXISTS update_playlists_modtime ON playlists;
        CREATE TRIGGER update_playlists_modtime
            BEFORE UPDATE ON playlists
            FOR EACH ROW
            EXECUTE FUNCTION update_modified_column();
    END IF;
END $$;