# CampusCompile: The Ultimate Collegiate Coding Arena
CampusCompile is a centralized, high-performance competitive programming platform built specifically for the university ecosystem. It bridges the gap between generic coding websites and the specific, rigorous demands of academic coursework, providing a secure arena for campus-wide coding competitions, algorithmic problem sets, and automated code evaluation.

# 🌟 The Vision
Generic platforms don't understand university structures. CampusCompile is engineered to map directly to campus demographics—allowing faculty to target contests to specific courses, graduation years, and student groups, all while enforcing strict academic integrity.

# 🚀 Core Features
## 🛡️ Ironclad Academic Integrity
* Proactive Telemetry: The frontend aggressively monitors and flags suspicious behavior, including tab-switching, focus loss, and unauthorized copy-pasting.

* Auto-Typer Detection: Analyzes keystroke variance and WPM in real-time to flag artificially injected code.

* Environment Lockout: Forces a secure fullscreen environment during live contests; exiting triggers a recorded security lockout.

* Automated MOSS Audits: The backend automatically bundles contest submissions and interfaces with Stanford's Measure of Software Similarity (MOSS) to generate deep structural plagiarism reports post-contest.

## 🎯 Granular Demographic Access
Faculty can restrict contest access using precise rules: Allowed Courses (e.g., B.Tech, MCA), Departments (e.g., CSE, ECE), Batches, Sections, and Graduation Years.

## ⚡ Precision Execution Engine
* Sandboxed Evaluation: Code is executed in fully isolated, ephemeral Docker containers with strict, dynamically allocated CPU, memory, and PID limits.

* Real-Time Streaming: Students receive instant, character-by-character feedback via Server-Sent Events (SSE) as their code runs against hidden test cases.

* Language Support: Natively supports C++ 20, Python 3, and Java 17, with automatic time and memory limit multipliers applied based on the language's inherent overhead.

## 👥 The User Experience
### For Faculty & Admins (The Forge)
* Rich Problem Creation: Craft algorithmic challenges using a dual-pane Markdown editor with full LaTeX support for mathematical formulas.

* Frictionless Test Cases: Upload massive I/O test cases via a simple drag-and-drop ZIP extraction tool, which are automatically streamed securely to S3 storage.

* Live Overseer Dashboard: Monitor live leaderboards enriched with real-time telemetry alerts, showing exactly which students are triggering integrity flags.

### For Students (The Arena)
* Microsoft SSO: Seamless one-click login utilizing the university's existing Azure Active Directory.

* Professional Workspace: A distraction-free coding environment powered by the Monaco Editor (the engine behind VS Code).

# 🏗️ Tech Stack & Engineering Choices
* CampusCompile utilizes a microservices-oriented architecture to ensure high availability, security, and scalability during massive campus-wide events.

* Backend API — Go (Golang): Chosen for its lightning-fast concurrency model. Go effortlessly handles hundreds of simultaneous Server-Sent Events (SSE) connections for live leaderboards and submission tracking without choking the server's memory.

* Execution Worker — Python: Ideal for scripting complex sandboxed environments. Python seamlessly interfaces with the Docker daemon to spawn isolated compilation containers and handles the complex network requests required for Stanford's MOSS anti-plagiarism API.

* Frontend — React + Vite: Delivers a buttery-smooth Single Page Application experience, which is absolutely critical for embedding the heavy Monaco Code Editor and maintaining rapid state changes during intense coding contests.

* Primary Database — PostgreSQL: Provides strict ACID compliance and relational integrity. Essential for mapping complex demographic rules, user roles, and historical submission logs.

* Message Broker — Redis: Acts as the high-throughput nervous system of the platform. It queues student submissions for the Python workers and manages the Pub/Sub channels that broadcast live execution verdicts back to the Go API.

* Object Storage — S3 / MinIO: Ensures isolated, highly scalable, and secure storage for massive test-case ZIP files and user source code, keeping the core database lightweight.