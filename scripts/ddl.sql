-- ==========================================
-- 1. CUSTOM TYPES
-- ==========================================
CREATE TYPE user_role AS ENUM ('student', 'professor', 'admin');
CREATE TYPE problem_difficulty AS ENUM ('Easy', 'Medium', 'Hard');
CREATE TYPE submission_status AS ENUM ('Pending', 'Running', 'AC', 'WA', 'TLE', 'MLE', 'RE', 'CE', 'SE');
CREATE TYPE telemetry_event_type AS ENUM ('blur', 'paste_attempt', 'autotyper_suspected', 'visibility_spoof_suspected', 'anomalous_routing');
CREATE TYPE audit_status AS ENUM ('pending', 'in_progress', 'completed', 'failed');

-- ==========================================
-- 2. CORE ENTITIES
-- ==========================================

-- Users table: Stores all user profiles and authentication mappings
CREATE TABLE users (
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
CREATE TABLE problems (
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
    is_public BOOLEAN DEFAULT false
);

-- Contests table: Manages competition windows and rules
CREATE TABLE contests (
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
    moss_audit_status audit_status DEFAULT 'pending'
);

-- ==========================================
-- 3. RELATIONAL & ACTIVITY TABLES
-- ==========================================

-- Contest Registrations: Maps users to the contests they joined
CREATE TABLE contest_registrations (
    contest_id UUID REFERENCES contests(contest_id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    registered_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (contest_id, user_id)
);

-- Contest Problems: Maps specific problems to specific contests
CREATE TABLE contest_problems (
    contest_id UUID REFERENCES contests(contest_id) ON DELETE CASCADE,
    problem_id UUID REFERENCES problems(problem_id) ON DELETE CASCADE,
    points_value INTEGER NOT NULL DEFAULT 100,
    PRIMARY KEY (contest_id, problem_id)
);

-- Submissions table: Tracks code execution runs
CREATE TABLE submissions (
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
CREATE TABLE test_cases (
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
CREATE TABLE contest_telemetry (
    telemetry_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id UUID REFERENCES contests(contest_id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    event_type telemetry_event_type NOT NULL,
    metadata JSONB, 
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Plagiarism Reports: Stores MOSS audit results from the Python worker
CREATE TABLE plagiarism_reports (
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
CREATE INDEX idx_contests_author_id ON contests(author_id);
CREATE INDEX idx_contests_times ON contests(start_time, end_time);
CREATE INDEX idx_contests_moss_audit ON contests(moss_audit_status, end_time, updated_at);

-- Contest Registrations
CREATE INDEX idx_contest_registrations_user_id ON contest_registrations(user_id);

-- Submissions
CREATE INDEX idx_submissions_contest_id ON submissions(contest_id);
CREATE INDEX idx_submissions_problem_id ON submissions(problem_id);
CREATE INDEX idx_submissions_user_id ON submissions(user_id);
CREATE INDEX idx_submissions_status ON submissions(status);

-- Telemetry & Plagiarism
CREATE INDEX idx_telemetry_contest_user ON contest_telemetry(contest_id, user_id);
CREATE INDEX idx_plagiarism_contest_problem ON plagiarism_reports(contest_id, problem_id);


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
CREATE TRIGGER update_contests_modtime
    BEFORE UPDATE ON contests
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();