-- 1. Create Custom ENUM Types for strict data enforcement
CREATE TYPE user_role AS ENUM ('student', 'professor', 'admin');
CREATE TYPE problem_difficulty AS ENUM ('Easy', 'Medium', 'Hard');
CREATE TYPE submission_status AS ENUM ('Pending', 'Running', 'AC', 'WA', 'TLE', 'MLE', 'RE', 'CE');

-- 2. Create the Users Table
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
    is_onboarded BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- 3. Create the Problems Table
CREATE TABLE problems (
    -- Primary Identification
    problem_id      uuid DEFAULT gen_random_uuid() NOT NULL,
    polygon_id      text, -- Unique ID from the Polygon package
    title           varchar(255) NOT NULL,
    slug            varchar(255) NOT NULL,
    
    -- Content and Metadata
    description     text NOT NULL,
    difficulty      public."problem_difficulty" NOT NULL,
    
    -- Resource Constraints (Standardized to Polygon Units)
    time_limit_ms   int4 DEFAULT 2000 NOT NULL,
    memory_limit_kb int4 DEFAULT 262144 NOT NULL,
    
    -- Modality C: Specialized Judging
    has_checker     boolean DEFAULT FALSE, -- True if problem uses a checker.cpp
    checker_path    text, -- Storage path for the compiled checker
    
    -- Ownership and Timestamps
    author_id       uuid NULL,
    created_at      timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
    
    -- Constraints
    CONSTRAINT problems_pkey PRIMARY KEY (problem_id),
    CONSTRAINT problems_slug_key UNIQUE (slug)
);

-- 4. Create the Test Cases Table
CREATE TABLE test_cases (
    -- Primary Identification
    test_case_id      uuid DEFAULT gen_random_uuid() NOT NULL,
    problem_id        uuid NOT NULL, -- Changed from NULL to NOT NULL for strict relation
    
    -- Ordering and Visibility
    test_index        int4 NOT NULL, -- Required to maintain Polygon's 1, 2, 3 sequence
    is_hidden         boolean DEFAULT true NOT NULL, -- Maps to Polygon: sample="true" means is_hidden=false
    
    -- Scalable File System Pointers (Replaces input_data & expected_output)
    input_file_path   text NOT NULL,
    output_file_path  text NOT NULL,
    
    -- Advanced Judging (Optional but highly recommended)
    output_hash       varchar(64), -- SHA-256 hash of the output for lightning-fast wrong-answer rejections
    
    -- Constraints
    CONSTRAINT test_cases_pkey PRIMARY KEY (test_case_id),
    CONSTRAINT test_cases_problem_id_fkey FOREIGN KEY (problem_id) 
        REFERENCES problems(problem_id) 
        ON DELETE CASCADE
);

-- 5. Create the Submissions Table
CREATE TABLE submissions (
    submission_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    problem_id UUID REFERENCES problems(problem_id) ON DELETE CASCADE,
    source_code TEXT NOT NULL,
    language VARCHAR(50) NOT NULL,
    status submission_status DEFAULT 'Pending',
    error_logs TEXT, -- 👇 NEW: Stores the cc1plus or javac errors
    execution_time_ms INT,
    memory_used_kb INT,
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 6. Create the Contests Table
CREATE TABLE contests (
    contest_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    host_organization VARCHAR(255),
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 7. Create the Contest Problems Mapping Table
CREATE TABLE contest_problems (
    contest_id UUID REFERENCES contests(contest_id) ON DELETE CASCADE,
    problem_id UUID REFERENCES problems(problem_id) ON DELETE CASCADE,
    points_value INT NOT NULL DEFAULT 100,
    PRIMARY KEY (contest_id, problem_id)
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