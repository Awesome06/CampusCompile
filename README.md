# CampusCompile — Scalable Collegiate Coding Arena

A centralized, high-performance competitive programming platform built specifically for the university ecosystem. CampusCompile bridges the gap between generic coding websites and rigorous academic coursework by providing a secure, distributed arena for campus-wide algorithmic competitions and automated code evaluation.

## Contents
- [Quick start](#quick-start)
- [Tech stack](#tech-stack)
- [System Architecture](#system-architecture)
- [Core Features & Integrity](#core-features--integrity)
- [Database schema](#database-schema)
- [Execution Pipeline](#execution-pipeline)
- [Project layout](#project-layout)

## Quick start
Because CampusCompile relies on a microservices architecture (Go, Python, Redis, Postgres, MinIO), the entire platform is orchestrated via Docker Compose.

Requires **Docker** and **Docker Compose**.

```bash
# Clone the repository
git clone [https://github.com/Awesome06/CampusCompile.git](https://github.com/Awesome06/CampusCompile.git)
cd CampusCompile

# Build and spin up the microservices
docker-compose up --build
```
* **Student/Admin SPA:** `http://localhost:3000`
* **Go API Gateway:** `http://localhost:8080`
* **MinIO Object Storage Console:** `http://localhost:9001`

## Tech stack

| Component | Technology | Purpose |
| :--- | :--- | :--- |
| **Frontend** | React, Vite, Monaco Editor | High-performance SPA handling intense live state changes. |
| **API Gateway** | Go (Golang) | Lightning-fast concurrency for handling hundreds of SSE connections. |
| **Execution Worker** | Python | Orchestrates Docker daemon for sandboxing and interfaces with Stanford MOSS. |
| **Message Broker** | Redis | High-throughput Pub/Sub and async queueing for student submissions. |
| **Primary Database**| PostgreSQL | Strict ACID compliance for complex demographic mapping and roles. |
| **Object Storage** | S3 / MinIO | Scalable storage for massive test-case ZIP files and user source code. |

## System Architecture

```text
┌─────────────────────────┐        ┌─────────────────────────┐
│  React SPA (The Arena)  │◄──────►│ Microsoft Azure AD SSO  │
└────────────┬────────────┘        └─────────────────────────┘
             │ HTTP / SSE
┌────────────▼────────────┐        ┌─────────────────────────┐
│      Go API Gateway     │◄──────►│       PostgreSQL        │
└────────────┬────────────┘        └─────────────────────────┘
             │ Enqueues Job
┌────────────▼────────────┐        ┌─────────────────────────┐
│   Redis Message Broker  │◄──────►│      MinIO / AWS S3     │
└────────────┬────────────┘        └─────────────────────────┘
             │ Consumes Job
┌────────────▼────────────┐        ┌─────────────────────────┐
│  Python Grading Worker  │───────►│ Stanford MOSS (Audits)  │
└────────────┬────────────┘        └─────────────────────────┘
             │ Spawns Sandbox
┌────────────▼────────────┐
│   Docker (C++/Py/Java)  │
└─────────────────────────┘
```

Three core engineering decisions define the platform:
1. **Asynchronous Execution:** Code submission is completely decoupled from evaluation. The Go API ingests submissions and pushes them to Redis. Python workers pick up the jobs, compile/execute in Docker, and broadcast verdicts back to the client via Go Server-Sent Events (SSE). This ensures the API never drops requests during peak traffic spikes.
2. **Ephemeral Sandboxing:** Every student submission is executed in a fully isolated, ephemeral Docker container with strict, dynamically allocated CPU, memory, and PID limits.
3. **Academic-First Access Control:** Unlike generic platforms, CampusCompile's RBAC is mapped directly to university hierarchies (Courses, Departments, Batches, Sections, Graduation Years).

## Core Features & Integrity

To ensure zero-trust academic integrity during campus-wide exams, CampusCompile enforces strict telemetry and security measures:

| Feature | Mechanism |
| :--- | :--- |
| **Proactive Telemetry** | The frontend aggressively monitors and flags suspicious behavior, including tab-switching, focus loss, and unauthorized copy-pasting. |
| **Auto-Typer Detection** | Analyzes keystroke variance and WPM in real-time to flag artificially injected code. |
| **Environment Lockout** | Forces a secure fullscreen environment during live contests; exiting triggers a recorded security lockout. |
| **Stanford MOSS Audits** | The backend automatically bundles contest submissions and sends them to Stanford's Measure of Software Similarity (MOSS) API to generate deep structural plagiarism reports post-contest. |
| **Language Overhead Scaling**| Natively supports C++20, Python 3, and Java 17, automatically applying time/memory limit multipliers to account for runtime overhead (e.g., JVM startup vs. compiled C++). |

## Database schema
A simplified view of the core relational structures enforcing demographic rules and contest state.

```text
users
├─ id PK
├─ azure_oid UNIQUE
├─ email
├─ role 'student' | 'faculty' | 'admin'
├─ department
├─ batch_year
└─ section

contests
├─ id PK
├─ title
├─ start_time
├─ end_time
├─ allowed_departments ARRAY
└─ allowed_batches ARRAY

problems
├─ id PK
├─ contest_id ─► contests.id
├─ title
├─ description (Markdown/LaTeX)
├─ time_limit_ms
└─ memory_limit_mb

submissions
├─ id PK
├─ user_id ─► users.id
├─ problem_id ─► problems.id
├─ language
├─ status 'pending' | 'running' | 'accepted' | 'rejected'
├─ telemetry_flags JSONB
└─ submitted_at
```

## Execution Pipeline

The core workflow relies on a mix of REST endpoints and persistent connections:

1. **Submission (`POST /api/submit`):** The React client posts source code. The Go API writes a `pending` record to PostgreSQL, uploads the code to MinIO, and pushes a job ID to Redis.
2. **Evaluation (Python Worker):** A worker pops the job from Redis, pulls the code and test cases from MinIO, and spins up a restricted Docker container.
3. **Streaming Feedback (`GET /api/stream/submission/{id}`):** The React client opens an SSE connection to the Go API. As the Python worker evaluates test cases, it publishes verdicts to a Redis Pub/Sub channel. The Go API relays these directly to the client, providing character-by-character real-time feedback.

## Project layout

```text
.
├── backend-api/          # Go API Gateway (Fiber/Gin, PostgreSQL, Redis Pub/Sub)
├── execution-worker/     # Python Worker (Docker SDK, MinIO Client, MOSS API)
├── frontend/             # React SPA (Monaco Editor, SSE hooks, Telemetry)
├── docker-compose.yml    # Full local environment orchestration
├── scripts/              # DB migrations and test-case seeder scripts
└── README.md
```

## Contact
**Mrigank Bhatnagar**  
* [LinkedIn](https://linkedin.com/in/mrigank-bhatnagar)
* [GitHub](https://github.com/Awesome06)
