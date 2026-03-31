# Worker (Execution Platform) Agent Rules

## Tech Stack & Key Libraries
* **Language**: Python 3
* **Containerization**: `docker` (Python Docker SDK)
* **Queue / Message Broker**: `redis`
* **Database Driver**: `psycopg2-binary`
* **Cloud / File Storage**: `boto3` (S3 interactions)
* **Anti-plagiarism**: `mosspy` (Stanford MOSS API wrapper), `beautifulsoup4`, `requests`

## Architecture & Conventions
* **`queue_listener.py`**: Listens and pops submission payloads off the Redis queue. Must run reliably without memory leaks.
* **`runner.py`**: Responsible for spinning up ephemeral Docker containers, mounting code/tests, compiling/running, and aggregating outputs.
* **`moss_auditor.py`**: Logic for bundling directory assets and transmitting them securely to Stanford's MOSS service for codebase overlap analysis.

## Style & Best Practices
* **Sandboxed Evaluation**: All user-submitted code MUST be executed in strictly isolated, ephemeral Docker containers. You must dynamically enforce CPU limits, RAM constraints, and PID allocations.
* **Time & Memory Bounds**: Apply automatic time and memory limit multipliers based on requested execution language overhead (e.g., slower times for Python vs native C++ speed). 
* **Resilience**: Handle missing Docker daemons gracefully. Use precise timeout and retry logic for container execution loops and external network bounds (MOSS, S3).
* **Logging/Monitoring**: Keep standard output parsing and crash logging explicit. Clean up containers robustly to avoid creating "zombie" processes taking up host memory.

## Developer Workflow
* **Run Worker Queue Listener**: `python queue_listener.py`
