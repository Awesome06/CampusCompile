# CampusCompile

CampusCompile is the ultimate collegiate coding arena. Built specifically for the university ecosystem, it is a centralized platform designed to host campus-wide coding competitions, store algorithmic problem sets, and automatically evaluate student code with precision.

## Features

*   **Competition Management:** Host and manage campus-wide coding contests.
*   **Problem Repository:** Centralized storage for algorithmic challenges.
*   **Automated Evaluation:** Secure, sandboxed code execution engine.
*   **Real-time Feedback:** Instant results for student submissions.

## Tech Stack

*   **Frontend:** React (Vite)
*   **Backend:** Go (Golang)
*   **Code Execution Engine:** Python
*   **Database:** PostgreSQL
*   **Message Queue:** Redis
*   **Infrastructure:** Docker & Docker Compose

## Architecture

The platform consists of several containerized services orchestrated via Docker Compose:

1.  **Frontend (`campus-frontend`)**: The user interface for students and administrators, running on port `5173`.
2.  **API (`campus-api`)**: The core backend service written in Go, exposing REST endpoints on port `8080`.
3.  **Worker (`campus-worker`)**: A Python worker that listens to Redis queues to execute student code in a sandboxed environment. It has access to the Docker socket to spawn isolated containers for code execution.
4.  **Database (`campus-db`)**: A PostgreSQL 15 database for persistent storage.
5.  **Redis (`campus-redis`)**: Used as a message broker for asynchronous code execution tasks.

## Getting Started

### Prerequisites

*   Docker Desktop installed on your machine.

### Installation & Running

1.  Clone the repository.
2.  Navigate to the project directory.
3.  Start the application using Docker Compose:

    ```bash
    docker-compose up --build
    ```

4.  To stop the application:

    ```bash
    docker-compose down
    ```

### Accessing Services

*   **Web Interface:** http://localhost:5173
*   **API Endpoint:** http://localhost:8080
*   **Database:** Port `5433` (mapped from container port 5432)
