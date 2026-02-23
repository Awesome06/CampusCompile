-- Step 1 - Clean
-- Drop the old database completely (This deletes all tables, including the users table)
DROP DATABASE IF EXISTS "CampusCompile_db";

-- Drop any previous database users you may have created
DROP USER IF EXISTS campus_admin;
DROP USER IF EXISTS campus_app;

-- Step 2 - Set up
-- 1. Create the Database Owner (Admin)
CREATE USER campus_admin WITH PASSWORD 'admin_password_here' CREATEDB;

-- 2. Create the Application User (For your backend API)
CREATE USER campus_app WITH PASSWORD 'app_password_here';

-- 3. Create the fresh Database and assign ownership to the Admin
CREATE DATABASE "CampusCompile_db" OWNER campus_admin;

-- 4. Grant connection privileges to the app user
GRANT CONNECT ON DATABASE "CampusCompile_db" TO campus_app;

-- Step 3 - Permissions
-- Grant the app user permission to use the default schema
GRANT USAGE ON SCHEMA public TO campus_app;

-- Grant Data Manipulation Language (DML) rights to the app user
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO campus_app;

-- Ensure future tables created by the admin automatically grant permissions to the app user
ALTER DEFAULT PRIVILEGES IN SCHEMA public 
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO campus_app;