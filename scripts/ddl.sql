-- 1. Create Custom ENUM Types for strict data enforcement
CREATE TYPE user_role AS ENUM ('student', 'professor', 'admin');
CREATE TYPE problem_difficulty AS ENUM ('Easy', 'Medium', 'Hard');
CREATE TYPE submission_status AS ENUM ('Pending', 'Running', 'AC', 'WA', 'TLE', 'MLE', 'RE', 'CE');

-- 2. Create the Users Table
CREATE TABLE users (
    user_id         UUID               NOT NULL DEFAULT gen_random_uuid(),
    provider_id     VARCHAR(255)       NULL,
    email           VARCHAR(255)       NOT NULL,
    username        VARCHAR(50)        NULL,
    real_name       VARCHAR(255)       NULL,
    batch           VARCHAR(50)        NULL,
    "section"       VARCHAR(50)        NULL,
    student_group   VARCHAR(50)        NULL,
    course          VARCHAR(100)       NULL,
    department      VARCHAR(100)       NULL,
    graduation_year INTEGER            NULL,
    "role"          "user_role"        NOT NULL DEFAULT 'student'::user_role,
    campus_rating   INTEGER            NULL     DEFAULT 1200,
    is_onboarded    BOOLEAN            NULL     DEFAULT FALSE,
    created_at      TIMESTAMPTZ        NULL     DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT users_pkey 
        PRIMARY KEY (user_id),

    CONSTRAINT users_email_key 
        UNIQUE (email),

    CONSTRAINT users_provider_id_key 
        UNIQUE (provider_id),

    CONSTRAINT users_username_key 
        UNIQUE (username)
);

-- 3. Create the Problems Table
CREATE TABLE problems (
    problem_id        UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    title             VARCHAR(255) NOT NULL,
    slug              VARCHAR(255) NOT NULL UNIQUE,
    description       TEXT NOT NULL,
    difficulty        public.problem_difficulty NOT NULL,
    time_limit_ms     INTEGER DEFAULT 2000 NOT NULL,
    memory_limit_kb   INTEGER DEFAULT 262144 NOT NULL,
    author_id         UUID, REFERENCES users(user_id) ON DELETE SET NULL
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    has_checker       BOOLEAN DEFAULT FALSE,
    checker_s3_key    TEXT,
    is_public         BOOLEAN DEFAULT FALSE
);

-- 4. Create the Test Cases Table
CREATE TABLE test_cases (
    test_case_id    UUID         NOT NULL DEFAULT gen_random_uuid(),
    problem_id      UUID         NULL,
    input_data      TEXT         NULL,
    expected_output TEXT         NULL,
    is_hidden       BOOLEAN      NULL     DEFAULT TRUE,
    input_s3_key    VARCHAR(512) NULL,
    expected_s3_key VARCHAR(512) NULL,

    CONSTRAINT test_cases_pkey 
        PRIMARY KEY (test_case_id),

    CONSTRAINT test_cases_problem_id_fkey 
        FOREIGN KEY (problem_id) 
        REFERENCES public.problems (problem_id) 
        ON DELETE CASCADE
);

-- 5. Create the Submissions Table
CREATE TABLE submissions (
    submission_id     UUID                       NOT NULL DEFAULT gen_random_uuid(),
    user_id           UUID                       NULL,
    problem_id        UUID                       NULL,
    source_code       TEXT                       NOT NULL,
    "language"        VARCHAR(50)                NOT NULL,
    status            "submission_status"        NULL     DEFAULT 'Pending'::submission_status,
    execution_time_ms INTEGER                    NULL,
    memory_used_kb    INTEGER                    NULL,
    submitted_at      TIMESTAMPTZ                NULL     DEFAULT CURRENT_TIMESTAMP,
    error_logs        TEXT                       NULL,

    CONSTRAINT submissions_pkey 
        PRIMARY KEY (submission_id),

    CONSTRAINT submissions_problem_id_fkey 
        FOREIGN KEY (problem_id) 
        REFERENCES public.problems (problem_id) 
        ON DELETE CASCADE
);

-- 6. Create the Contests Table
CREATE TABLE contests (
    contest_id        UUID         NOT NULL DEFAULT gen_random_uuid(),
    title             VARCHAR(255) NOT NULL,
    host_organization VARCHAR(255) NULL,
    start_time        TIMESTAMPTZ  NOT NULL,
    end_time          TIMESTAMPTZ  NOT NULL,
    created_at        TIMESTAMPTZ  NULL     DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT contests_pkey 
        PRIMARY KEY (contest_id)
);

-- 7. Create the Contest Problems Mapping Table
CREATE TABLE contest_problems (
    contest_id   UUID    NOT NULL,
    problem_id   UUID    NOT NULL,
    points_value INTEGER NOT NULL DEFAULT 100,

    CONSTRAINT contest_problems_pkey 
        PRIMARY KEY (contest_id, problem_id),

    CONSTRAINT contest_problems_contest_id_fkey 
        FOREIGN KEY (contest_id) 
        REFERENCES contests (contest_id) 
        ON DELETE CASCADE,

    CONSTRAINT contest_problems_problem_id_fkey 
        FOREIGN KEY (problem_id) 
        REFERENCES problems (problem_id) 
        ON DELETE CASCADE
);

-- Optimize queries looking for a specific user's submissions
CREATE INDEX idx_submissions_user_id ON submissions(user_id);

-- Optimize queries loading all submissions for a specific problem
CREATE INDEX idx_submissions_problem_id ON submissions(problem_id);

-- Optimize queries filtering by AC, WA, TLE, etc. (Great for analytics)
CREATE INDEX idx_submissions_status ON submissions(status);

-- Optimize leaderboards for contests
CREATE INDEX idx_contests_times ON contests(start_time, end_time);

-- Index to instantly fetch all test cases for a specific problem when judging
CREATE INDEX idx_test_cases_problem_id ON test_cases(problem_id);

-- Index to instantly fetch ONLY the public samples for the React frontend
CREATE INDEX idx_test_cases_samples ON test_cases(problem_id, is_hidden) WHERE is_hidden = false;