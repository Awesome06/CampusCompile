import docker
import os
import shutil
import itertools
import boto3
from typing import Optional

# --- CONFIGURATION ---
client = docker.from_env()
SANDBOX_BASE = "/sandbox_shared"
CACHE_BASE = os.path.join(SANDBOX_BASE, "cache") # Local test case cache

s3_client = boto3.client(
    's3',
    endpoint_url=os.getenv('S3_ENDPOINT', 'http://minio:9000'),
    aws_access_key_id=os.getenv('S3_ACCESS_KEY', 'campus_admin'),
    aws_secret_access_key=os.getenv('S3_SECRET_KEY', 'campus_password'),
    region_name='us-east-1' 
)
BUCKET_NAME = "campus-testcases"

# --- CACHING & I/O HELPERS ---
def fetch_from_s3(s3_key: str, destination_path: str):
    try:
        s3_client.download_file(BUCKET_NAME, s3_key, destination_path)
    except Exception as e:
        raise Exception(f"Failed to fetch test case from S3: {str(e)}")

def ensure_cached_testcase(work_dir, problem_id, tc_id, input_data, expected_output, input_s3_key, expected_s3_key):
    """Downloads test cases to a persistent cache directory so they are only fetched once."""
    if str(problem_id) == "custom":
        tc_dir = work_dir # Custom runs aren't permanently cached
    else:
        tc_dir = os.path.join(CACHE_BASE, str(problem_id), str(tc_id))
        os.makedirs(tc_dir, exist_ok=True)
        
    input_path = os.path.join(tc_dir, 'input.txt')
    expected_path = os.path.join(tc_dir, 'expected.txt')
    
    # Only write/download if it doesn't exist in the cache
    if str(problem_id) == "custom" or not os.path.exists(input_path):
        if input_s3_key:
            fetch_from_s3(input_s3_key, input_path)
        else:
            with open(input_path, 'w', encoding='utf-8') as f: f.write(input_data or "")
                
    if str(problem_id) == "custom" or not os.path.exists(expected_path):
        if expected_s3_key:
            fetch_from_s3(expected_s3_key, expected_path)
        else:
            with open(expected_path, 'w', encoding='utf-8') as f: f.write(expected_output or "")
                
    return input_path, expected_path

def read_file_safely(filepath: str, max_chars: int = 4096) -> str:
    if not os.path.exists(filepath): return ""
    with open(filepath, 'r', encoding='utf-8', errors='replace') as f: return f.read(max_chars)

# --- EXECUTION STAGES ---
def compile_code(language: str, work_dir: str, source_code: str):
    """Compiles code exactly ONCE before looping through test cases."""
    if language == 'python':
        with open(os.path.join(work_dir, 'solution.py'), 'w', encoding='utf-8') as f: f.write(source_code)
        return None 

    elif language == 'cpp':
        with open(os.path.join(work_dir, 'solution.cpp'), 'w', encoding='utf-8') as f: f.write(source_code)
        container = client.containers.run(
            image="campus-cpp", command='g++ -O2 -w solution.cpp -o solution.out', 
            volumes={'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'rw'}},
            working_dir=work_dir, detach=True, network_disabled=True, mem_limit="512m"
        )
        res = container.wait(timeout=10)
        if res['StatusCode'] != 0:
            logs = container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
            container.remove(force=True)
            return {"verdict": "CE", "message": logs.strip()}
        container.remove(force=True)
        return None

    elif language == 'java':
        with open(os.path.join(work_dir, 'Main.java'), 'w', encoding='utf-8') as f: f.write(source_code)
        container = client.containers.run(
            image="campus-java", command='javac Main.java', 
            volumes={'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'rw'}},
            working_dir=work_dir, detach=True, network_disabled=True, mem_limit="512m"
        )
        res = container.wait(timeout=10)
        if res['StatusCode'] != 0:
            logs = container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
            container.remove(force=True)
            return {"verdict": "CE", "message": logs.strip()}
        container.remove(force=True)
        return None
        
    return {"verdict": "CE", "message": "Unsupported language"}

def run_code(language: str, work_dir: str, cached_input_path: str, time_limit_seconds: float, memory_limit_kb: int):
    """Executes the pre-compiled code using the referenced cached input file."""
    volumes = {'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'rw'}}
    
    # Notice we pass the cached file reference directly into the shell command
    if language == 'python':
        cmd = f'sh -c "python solution.py < {cached_input_path} > output.txt 2> error.txt"'
        img = "campus-python"
    elif language == 'cpp':
        cmd = f'sh -c "./solution.out < {cached_input_path} > output.txt 2> error.txt"'
        img = "campus-cpp"
    elif language == 'java':
        cmd = f'sh -c "java -Xmx256m Main < {cached_input_path} > output.txt 2> error.txt"'
        img = "campus-java"

    try:
        container = client.containers.run(
            image=img, command=cmd, volumes=volumes, working_dir=work_dir, 
            detach=True, network_disabled=True, mem_limit=f"{memory_limit_kb}k",
            cpu_period=100000, cpu_quota=100000, pids_limit=64, cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )
        result = container.wait(timeout=time_limit_seconds)
        container.remove(force=True)
        return {"status": "Success" if result['StatusCode'] == 0 else "Runtime Error"}
    except Exception as e:
        if "Timeout" in str(e) or "Read timed out" in str(e): return {"status": "Time Limit Exceeded"}
        return {"status": "System Error", "message": str(e)}

def evaluate_output(actual_output_file_path: str, cached_expected_path: str) -> str:
    def get_lines(path):
        if os.path.exists(path):
            with open(path, 'r', encoding='utf-8') as f:
                for line in f: yield line.rstrip()
        else:
            yield ""

    actual_iter = get_lines(actual_output_file_path)
    expected_iter = get_lines(cached_expected_path)

    sentinel = object()
    for a_line, e_line in itertools.zip_longest(actual_iter, expected_iter, fillvalue=sentinel):
        if a_line is sentinel:
            if e_line != "": return "WA"
            for remaining in expected_iter:
                if remaining != "": return "WA"
            break
        if e_line is sentinel:
            if a_line != "": return "WA"
            for remaining in actual_iter:
                if remaining != "": return "WA"
            break
        if a_line != e_line:
            return "WA"
    return "AC"

# --- THE MASTER GRADER ---
def grade_submission(submission_id: str, problem_id: str, language: str, source_code: Optional[str], source_s3_key: Optional[str], test_cases: list, time_limit_ms: int, memory_limit_kb: int):
    time_limit_sec = time_limit_ms / 1000.0
    work_dir = os.path.join(SANDBOX_BASE, str(submission_id))
    os.makedirs(work_dir, exist_ok=True)
    
    # 🛡️ HARDENING: Instead of 0o777, map to a specific UID inside the Alpine container
    # Assuming UID 1000 is used by a non-root user in your docker images
    os.chmod(work_dir, 0o750) 
    try:
        os.chown(work_dir, 1000, 1000) 
    except PermissionError:
        pass # Handle case where host running script isn't root
    
    try:
        # 👇 NEW: Fetch source code from S3 if a key is provided (Official submissions)
        if source_s3_key:
            file_ext = "cpp" if language == "cpp" else "py" if language == "python" else "java"
            download_path = os.path.join(work_dir, f"source.{file_ext}")
            
            try:
                fetch_from_s3(source_s3_key, download_path)
                with open(download_path, 'r', encoding='utf-8') as f:
                    source_code = f.read()
            except Exception as e:
                return {"verdict": "SE", "message": f"Failed to download source code: {str(e)}", "actual_output": ""}

        # Safety Check
        if not source_code:
            return {"verdict": "SE", "message": "Source code is empty.", "actual_output": ""}

        # Step 1: Compile Once
        compile_err = compile_code(language, work_dir, source_code)
        if compile_err:
            return compile_err 
            
        # Step 2: Stream through cached test cases
        for idx, tc in enumerate(test_cases):
            tc_id = tc.get('test_case_id', f"custom_{idx}")
            
            input_path, expected_path = ensure_cached_testcase(
                work_dir, problem_id, tc_id, 
                tc.get('input_data'), tc.get('expected_output'), 
                tc.get('input_s3_key'), tc.get('expected_s3_key')
            )
            
            run_result = run_code(language, work_dir, input_path, time_limit_sec, memory_limit_kb)
            
            if run_result["status"] == "Time Limit Exceeded":
                return {"verdict": "TLE", "message": f"Execution took too long on Test Case {idx+1}", "actual_output": ""}
            elif run_result["status"] == "Runtime Error":
                err_log = read_file_safely(os.path.join(work_dir, 'error.txt'))
                return {"verdict": "RE", "message": f"Runtime Error on Test Case {idx+1}.\n{err_log}", "actual_output": ""}
            elif run_result["status"] != "Success":
                return {"verdict": "SE", "message": run_result.get("message", "System Error"), "actual_output": ""}
                
            verdict = evaluate_output(os.path.join(work_dir, 'output.txt'), expected_path)
            if verdict != "AC":
                actual_out = read_file_safely(os.path.join(work_dir, 'output.txt'))
                return {"verdict": "WA", "message": f"Wrong Answer on Test Case {idx+1}", "actual_output": actual_out}
                
        return {"verdict": "AC", "actual_output": read_file_safely(os.path.join(work_dir, 'output.txt'))}
        
    except Exception as e:
        return {"verdict": "SE", "message": str(e), "actual_output": ""}
    finally:
        # Ensure cleanup runs even if the worker crashes
        shutil.rmtree(work_dir, ignore_errors=True)