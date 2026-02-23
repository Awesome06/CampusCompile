-- Table Test

SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
ORDER BY table_name;

-- Expected Output of 6 tables

-- Enum Test
SELECT typname, string_agg(enumlabel, ', ') as allowed_values
FROM pg_type t
JOIN pg_enum e ON t.oid = e.enumtypid
WHERE typname IN ('user_role', 'problem_difficulty', 'submission_status')
GROUP BY typname;

-- Expected Output of 3 rows with allowed string values

--Index Test
SELECT tablename, indexname, indexdef 
FROM pg_indexes 
WHERE schemaname = 'public' 
  AND indexname NOT LIKE '%_pkey' -- Hides default primary key indexes
ORDER BY tablename, indexname;

-- Expected output of 4 indexs starting with idx and other unique indexs ending with pk