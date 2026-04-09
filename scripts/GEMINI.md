# Scripts Agent Rules

## Tech Stack
* Environment: Standard shell / SQL / Makefile
* Current Context: Includes database DDLs (`ddl.sql`) and potentially other deployment/utility scripts.

## Style & Best Practices
* **Idempotency:** All bash, python, or SQL scripts must be idempotent (can run multiple times without causing side effects or failures). 
* **Comment Triggers:** DDL schemas should include clear cascading logic and appropriate IF NOT EXISTS blocks to avoid crashing initializations.
* **Security:** Never place secrets, passwords, or connection strings in open plaintext scripts. Assume scripts will run in varying development and production environments.
