-- 1. Create Custom ENUM Types for strict data enforcement
CREATE TYPE user_role AS ENUM ('student', 'professor', 'admin');
CREATE TYPE problem_difficulty AS ENUM ('Easy', 'Medium', 'Hard');
CREATE TYPE submission_status AS ENUM ('Pending', 'Running', 'AC', 'WA', 'TLE', 'MLE', 'RE', 'CE');

-- 2. Create the Users Table
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'student',
    campus_rating INT DEFAULT 1200,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Create the Problems Table
CREATE TABLE problems (
    problem_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT NOT NULL,
    difficulty problem_difficulty NOT NULL,
    time_limit_ms INT NOT NULL DEFAULT 2000, 
    memory_limit_kb INT NOT NULL DEFAULT 262144, -- Defaults to 256MB
    author_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Create the Test Cases Table
CREATE TABLE test_cases (
    test_case_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    problem_id UUID REFERENCES problems(problem_id) ON DELETE CASCADE,
    input_data TEXT NOT NULL,
    expected_output TEXT NOT NULL,
    is_hidden BOOLEAN DEFAULT TRUE
);

-- 5. Create the Submissions Table
CREATE TABLE submissions (
    submission_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    problem_id UUID REFERENCES problems(problem_id) ON DELETE CASCADE,
    source_code TEXT NOT NULL,
    language VARCHAR(50) NOT NULL,
    status submission_status DEFAULT 'Pending',
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