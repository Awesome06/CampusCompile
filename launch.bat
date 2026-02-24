@echo off
echo ==========================================
echo    Launching CampusCompile Environment
echo ==========================================

:: 1. CHECK AND START REDIS
echo [*] Checking Redis Server...
docker ps | findstr "campus-redis" >nul
if %errorlevel% neq 0 (
    echo [*] Redis is not running. Checking if container exists...
    docker ps -a | findstr "campus-redis" >nul
    if %errorlevel% neq 0 (
        echo [+] Creating and starting new Redis container...
        docker run --name campus-redis -p 6379:6379 -d redis
    ) else (
        echo [+] Starting existing Redis container...
        docker start campus-redis
    )
) else (
    echo [+] Redis is already running!
)

:: 2. START GO API
echo [*] Booting up Go API Server...
start "Go API Server" cmd /k "cd server && go run main.go"

:: 3. START PYTHON WORKER
:: (If your python files are in a folder like 'worker', change this to: cd worker && python queue_listener.py)
echo [*] Booting up Python Execution Worker...
start "Python Worker" cmd /k "cd worker && python queue_listener.py"

:: 4. START REACT FRONTEND
echo [*] Booting up React Frontend...
start "React Frontend" cmd /k "cd frontend && npm run dev"

echo ==========================================
echo [+] All systems nominal. Ready to code.
echo ==========================================
pause