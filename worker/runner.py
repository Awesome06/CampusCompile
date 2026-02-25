import docker
import os
import uuid
import shutil

# Initialize the Docker client
client = docker.from_env()

# Base directory for the shared Docker volume
SANDBOX_BASE = "/sandbox_shared"

# --- PYTHON ---
def execute_python_code(source_code: str, input_data: str, time_limit_seconds: float = 2):
    sub_id = str(uuid.uuid4())
    work_dir = os.path.join(SANDBOX_BASE, sub_id)
    os.makedirs(work_dir, exist_ok=True)
    os.chmod(work_dir, 0o777)
    
    try:
        with open(os.path.join(work_dir, 'solution.py'), 'w', encoding='utf-8') as f:
            f.write(source_code)
        with open(os.path.join(work_dir, 'input.txt'), 'w', encoding='utf-8') as f:
            f.write(input_data)

        container = client.containers.run(
            image="campus-python",
            command='sh -c "python solution.py < input.txt"',
            # 👇 MOUNTS THE SHARED NAMED VOLUME DIRECTLY
            volumes={'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'ro'}},
            working_dir=work_dir, 
            detach=True, network_disabled=True, mem_limit="256m", 
            cpu_period=100000, cpu_quota=100000, pids_limit=64, cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )

        result = container.wait(timeout=time_limit_seconds)
        exit_code = result['StatusCode']
        logs = container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
        container.remove(force=True)

        return {"status": "Success" if exit_code == 0 else "Runtime Error", "exit_code": exit_code, "output": logs.strip()}
    
    except Exception as e:
        if "Timeout" in str(e) or "Read timed out" in str(e):
            return {"status": "Time Limit Exceeded", "output": ""}
        return {"status": "System Error", "output": str(e)}
    finally:
        # Clean up the specific submission folder
        shutil.rmtree(work_dir, ignore_errors=True)

# --- CPP ---
def execute_cpp_code(source_code: str, input_data: str, time_limit_seconds: float = 2):
    sub_id = str(uuid.uuid4())
    work_dir = os.path.join(SANDBOX_BASE, sub_id)
    os.makedirs(work_dir, exist_ok=True)
    os.chmod(work_dir, 0o777)

    try:
        with open(os.path.join(work_dir, 'solution.cpp'), 'w', encoding='utf-8') as f:
            f.write(source_code)
        with open(os.path.join(work_dir, 'input.txt'), 'w', encoding='utf-8') as f:
            f.write(input_data)

        # COMPILATION
        compile_container = client.containers.run(
            image="campus-cpp",
            command='g++ -O2 -w solution.cpp -o solution.out',
            volumes={'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'rw'}},
            working_dir=work_dir,
            detach=True, network_disabled=True, mem_limit="512m", pids_limit=64, cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )
        compile_result = compile_container.wait(timeout=10)
        if compile_result['StatusCode'] != 0:
            logs = compile_container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
            compile_container.remove(force=True)
            return {"status": "Compilation Error", "output": logs.strip()}
        compile_container.remove(force=True)

        # EXECUTION
        run_container = client.containers.run(
            image="campus-cpp",
            command='sh -c "./solution.out < input.txt"',
            volumes={'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'ro'}},
            working_dir=work_dir,
            detach=True, network_disabled=True, mem_limit="256m", cpu_period=100000, cpu_quota=100000, pids_limit=64, cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )
        result = run_container.wait(timeout=time_limit_seconds)
        exit_code = result['StatusCode']
        logs = run_container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
        run_container.remove(force=True)

        return {"status": "Success" if exit_code == 0 else "Runtime Error", "exit_code": exit_code, "output": logs.strip()}

    except Exception as e:
        if "Timeout" in str(e) or "Read timed out" in str(e):
            return {"status": "Time Limit Exceeded", "output": ""}
        return {"status": "System Error", "output": str(e)}
    finally:
        shutil.rmtree(work_dir, ignore_errors=True)

# --- JAVA ---
def execute_java_code(source_code: str, input_data: str, time_limit_seconds: float = 2):
    sub_id = str(uuid.uuid4())
    work_dir = os.path.join(SANDBOX_BASE, sub_id)
    os.makedirs(work_dir, exist_ok=True)
    os.chmod(work_dir, 0o777)

    try:
        with open(os.path.join(work_dir, 'Main.java'), 'w', encoding='utf-8') as f:
            f.write(source_code)
        with open(os.path.join(work_dir, 'input.txt'), 'w', encoding='utf-8') as f:
            f.write(input_data)

        # COMPILATION
        compile_container = client.containers.run(
            image="campus-java",
            command='javac Main.java',
            volumes={'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'rw'}},
            working_dir=work_dir,
            detach=True, network_disabled=True, mem_limit="512m", pids_limit=64, cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )
        compile_result = compile_container.wait(timeout=10)
        if compile_result['StatusCode'] != 0:
            logs = compile_container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
            compile_container.remove(force=True)
            return {"status": "Compilation Error", "output": logs.strip()}
        compile_container.remove(force=True)

        # EXECUTION
        run_container = client.containers.run(
            image="campus-java",
            command='sh -c "java -Xmx256m Main < input.txt"',
            volumes={'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'ro'}},
            working_dir=work_dir,
            detach=True, network_disabled=True, mem_limit="512m", cpu_period=100000, cpu_quota=100000, pids_limit=64, cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )
        result = run_container.wait(timeout=time_limit_seconds)
        exit_code = result['StatusCode']
        logs = run_container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
        run_container.remove(force=True)

        return {"status": "Success" if exit_code == 0 else "Runtime Error", "exit_code": exit_code, "output": logs.strip()}

    except Exception as e:
        if "Timeout" in str(e) or "Read timed out" in str(e):
            return {"status": "Time Limit Exceeded", "output": ""}
        return {"status": "System Error", "output": str(e)}
    finally:
        shutil.rmtree(work_dir, ignore_errors=True)

# --- TASK 4: THE MASTER JUDGE ---
def evaluate_output(actual_output: str, expected_output: str) -> str:
    """
    Compares the student's output against the test case expected output.
    Strips trailing whitespaces and empty lines to be forgiving of minor formatting issues.
    """
    actual_lines = actual_output.strip().splitlines()
    expected_lines = expected_output.strip().splitlines()

    if len(actual_lines) != len(expected_lines):
        return "WA"

    for actual_line, expected_line in zip(actual_lines, expected_lines):
        if actual_line.rstrip() != expected_line.rstrip():
            return "WA"

    return "AC"

def grade_submission(language: str, source_code: str, input_data: str, expected_output: str, time_limit_ms: int = 2000):
    """
    The master function that wraps the execution and evaluation steps.
    Returns the final verdict for the database.
    """
    input_data = input_data.replace('\\n', '\n')
    expected_output = expected_output.replace('\\n', '\n')
    time_limit_sec = time_limit_ms / 1000.0

    # 1. Execute the code based on language
    if language == 'python':
        result = execute_python_code(source_code, input_data, time_limit_sec)
    elif language == 'cpp':
        result = execute_cpp_code(source_code, input_data, time_limit_sec)
    elif language == 'java':
        result = execute_java_code(source_code, input_data, time_limit_sec)
    else:
        return {"verdict": "CE", "message": "Unsupported language"}

    # 2. Handle System / Compilation / Time limits
    if result["status"] == "Compilation Error":
        return {"verdict": "CE", "message": result["output"]}
    elif result["status"] == "Time Limit Exceeded":
        return {"verdict": "TLE", "message": "Execution took too long"}
    elif result["status"] == "Runtime Error":
        return {"verdict": "RE", "message": "Non-zero exit code"}
    elif result["status"] == "System Error":
        return {"verdict": "SE", "message": "Internal Docker Error"}

    # 3. Evaluate the output for AC or WA
    actual_output = result["output"]
    verdict = evaluate_output(actual_output, expected_output)

    return {
        "verdict": verdict,
        "actual_output": actual_output
    }