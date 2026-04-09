# CampusCompile - Global Agent Rules

This is the global configuration file for Agent tools (like AI assistants and automation tools). These rules apply to the entire `CampusCompile` monorepo.

## Project Overview
CampusCompile is a centralized competitive programming platform built for the university ecosystem, enforcing strict academic integrity and providing sandboxed execution environments.

## Global Style & Restrictions
* **Microservices Architecture:** Maintain strong boundaries between the frontend, Go backend server, and Python execution worker.
* **Documentation:** All major new features, APIs, and complex algorithms must be documented.
* **Security & Integrity:** Never commit sensitive credentials or secrets. Avoid bypassing telemetry or MOSS anti-plagiarism checks.
* **Error Handling:** Avoid basic/brittle string-based error handling. Prefer typed sentinel errors and strict validation (e.g., UUID validation) across the stack to avoid unhandled crashes.

## Tech Stack Overview
* **Frontend:** React + Vite
* **Backend API:** Go (Golang)
* **Execution Worker:** Python
* **Database:** PostgreSQL
* **Message Broker:** Redis
* **Object Storage:** S3 / MinIO
