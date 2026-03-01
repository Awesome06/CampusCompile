import docker
import os
import uuid
import shutil
import itertools
import boto3
from typing import Optional

# --- CONFIGURATION ---
client = docker.from_env()
SANDBOX_BASE = "/sandbox_shared"

# Initialize MinIO (S3) Client
s3_client = boto3.client(
    's3',
    endpoint_url=os.getenv('S3_ENDPOINT', 'http://minio:9000'),
    aws_access_key_id=os.getenv('S3_ACCESS_KEY', 'campus_admin'),
    aws_secret_access_key=os.getenv('S3_SECRET_KEY', 'campus_password'),
    region_name='us-east-1' # Standard dummy region for MinIO
)
BUCKET_NAME = "campus-testcases"

# --- HELPER: FILE DOWNLOADER & COMMAND GENERATOR ---
def fetch_from_s3(s3_key: str, destination_path: str):
    """Downloads an object from MinIO/S3."""
    try:
        s3_client.download_file(BUCKET_NAME, s3_key, destination_path)
    except Exception as e:
        raise Exception(f"Failed to fetch test case from S3: {str(e)}")

def prepare_execution_environment(work_dir: str, input_data: Optional[str], input_s3_key: Optional[str], base_command: str):
    """
    Downloads S3 files directly into the sandbox directory 
    instead of relying on local host mounts.
    """
    volumes = {'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'rw'}}
    input_txt_path = os.path.join(work_dir, 'input.txt')

    if input_s3_key:
        fetch_from_s3(input_s3_key, input_txt_path)
    else:
        with open(input_txt_path, 'w', encoding='utf-8') as f:
            f.write(input_data or "")
            
    # The command is now unified! No complex volume bindings needed.
    run_cmd = f'sh -c "{base_command} < input.txt > output.txt 2> error.txt"'
    return volumes, run_cmd

def read_file_safely(filepath: str, max_chars: int = 4096) -> str:
    if not os.path.exists(filepath): return ""
    with open(filepath, 'r', encoding='utf-8', errors='replace') as f: return f.read(max_chars)

# --- PYTHON ---
def execute_python_code(work_dir: str, source_code: str, input_data: Optional[str], time_limit_seconds: float, memory_limit_kb: int, input_s3_key: Optional[str]):
    try:
        with open(os.path.join(work_dir, 'solution.py'), 'w', encoding='utf-8') as f: f.write(source_code)
        volumes, run_cmd = prepare_execution_environment(work_dir, input_data, input_s3_key, "python solution.py")

        container = client.containers.run(
            image="campus-python", command=run_cmd, volumes=volumes, working_dir=work_dir, 
            detach=True, network_disabled=True, mem_limit=f"{memory_limit_kb}k",
            cpu_period=100000, cpu_quota=100000, pids_limit=64, cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )

        result = container.wait(timeout=time_limit_seconds)
        container.remove(force=True)
        return {"status": "Success" if result['StatusCode'] == 0 else "Runtime Error"}
    
    except Exception as e:
        if "Timeout" in str(e) or "Read timed out" in str(e): return {"status": "Time Limit Exceeded"}
        return {"status": "System Error", "message": str(e)}

# --- CPP ---
def execute_cpp_code(work_dir: str, source_code: str, input_data: Optional[str], time_limit_seconds: float, memory_limit_kb: int, input_s3_key: Optional[str]):
    try:
        with open(os.path.join(work_dir, 'solution.cpp'), 'w', encoding='utf-8') as f: f.write(source_code)
        
        # Compilation...
        compile_container = client.containers.run(
            image="campus-cpp", command='g++ -O2 -w solution.cpp -o solution.out', volumes={'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'rw'}},
            working_dir=work_dir, detach=True, network_disabled=True, mem_limit="512m", 
            cpu_period=100000, cpu_quota=100000, pids_limit=64, cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )
        compile_result = compile_container.wait(timeout=10)
        if compile_result['StatusCode'] != 0:
            logs = compile_container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
            compile_container.remove(force=True)
            return {"status": "Compilation Error", "message": logs.strip()}
        compile_container.remove(force=True)

        volumes, run_cmd = prepare_execution_environment(work_dir, input_data, input_s3_key, "./solution.out")
        run_container = client.containers.run(
            image="campus-cpp", command=run_cmd, volumes=volumes, working_dir=work_dir,
            detach=True, network_disabled=True, mem_limit=f"{memory_limit_kb}k",
            cpu_period=100000, cpu_quota=100000, pids_limit=64, cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )
        result = run_container.wait(timeout=time_limit_seconds)
        run_container.remove(force=True)
        return {"status": "Success" if result['StatusCode'] == 0 else "Runtime Error"}

    except Exception as e:
        if "Timeout" in str(e) or "Read timed out" in str(e): return {"status": "Time Limit Exceeded"}
        return {"status": "System Error", "message": str(e)}

# --- JAVA ---
def execute_java_code(work_dir: str, source_code: str, input_data: Optional[str], time_limit_seconds: float, memory_limit_kb: int, input_s3_key: Optional[str]):
    try:
        with open(os.path.join(work_dir, 'Main.java'), 'w', encoding='utf-8') as f: f.write(source_code)
        
        # Compilation...
        compile_container = client.containers.run(
            image="campus-java", command='javac Main.java', volumes={'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'rw'}},
            working_dir=work_dir, detach=True, network_disabled=True, mem_limit="512m", 
            cpu_period=100000, cpu_quota=100000, pids_limit=64, cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )
        compile_result = compile_container.wait(timeout=10)
        if compile_result['StatusCode'] != 0:
            logs = compile_container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
            compile_container.remove(force=True)
            return {"status": "Compilation Error", "message": logs.strip()}
        compile_container.remove(force=True)

        volumes, run_cmd = prepare_execution_environment(work_dir, input_data, input_s3_key, "java -Xmx256m Main")
        run_container = client.containers.run(
            image="campus-java", command=run_cmd, volumes=volumes, working_dir=work_dir,
            detach=True, network_disabled=True, mem_limit=f"{memory_limit_kb}k",
            cpu_period=100000, cpu_quota=100000, pids_limit=64, cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )
        result = run_container.wait(timeout=time_limit_seconds)
        run_container.remove(force=True)
        return {"status": "Success" if result['StatusCode'] == 0 else "Runtime Error"}

    except Exception as e:
        if "Timeout" in str(e) or "Read timed out" in str(e): return {"status": "Time Limit Exceeded"}
        return {"status": "System Error", "message": str(e)}


# --- THE MASTER JUDGE ---
def evaluate_output(work_dir: str, actual_output_file_path: str, expected_output: Optional[str], expected_s3_key: Optional[str] = None) -> str:
    """
    Evaluates output by downloading expected data from S3 if needed,
    then streaming line-by-line to prevent OOM.
    """
    expected_txt_path = os.path.join(work_dir, 'expected.txt')

    def get_expected_lines():
        if expected_s3_key:
            fetch_from_s3(expected_s3_key, expected_txt_path)
            with open(expected_txt_path, 'r', encoding='utf-8') as f:
                for line in f: yield line.rstrip()
        else:
            for line in (expected_output or "").replace('\\n', '\n').splitlines(): yield line.rstrip()

    def get_actual_lines():
        if os.path.exists(actual_output_file_path):
            with open(actual_output_file_path, 'r', encoding='utf-8') as f:
                for line in f: yield line.rstrip()
        else:
            yield "" 

    actual_iter = get_actual_lines()
    expected_iter = get_expected_lines()

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


def grade_submission(language: str, source_code: str, input_data: Optional[str], expected_output: Optional[str], time_limit_ms: int = 2000, memory_limit_kb: int = 256000, input_s3_key: Optional[str] = None, expected_s3_key: Optional[str] = None):
    time_limit_sec = time_limit_ms / 1000.0
    sub_id = str(uuid.uuid4())
    work_dir = os.path.join(SANDBOX_BASE, sub_id)
    os.makedirs(work_dir, exist_ok=True)
    os.chmod(work_dir, 0o777)

    try:
        if language == 'python':
            result = execute_python_code(work_dir, source_code, input_data, time_limit_sec, memory_limit_kb, input_s3_key)
        elif language == 'cpp':
            result = execute_cpp_code(work_dir, source_code, input_data, time_limit_sec, memory_limit_kb, input_s3_key)
        elif language == 'java':
            result = execute_java_code(work_dir, source_code, input_data, time_limit_sec, memory_limit_kb, input_s3_key)
        else:
            return {"verdict": "CE", "message": "Unsupported language", "actual_output": ""}

        output_txt_path = os.path.join(work_dir, 'output.txt')
        error_txt_path = os.path.join(work_dir, 'error.txt')
        
        if result["status"] == "Compilation Error":
            return {"verdict": "CE", "message": result.get("message", "Compilation Error"), "actual_output": ""}
        elif result["status"] == "Time Limit Exceeded":
            return {"verdict": "TLE", "message": "Execution took too long", "actual_output": ""}
        elif result["status"] == "Runtime Error":
            err_log = read_file_safely(error_txt_path)
            if not err_log: err_log = read_file_safely(output_txt_path) 
            return {"verdict": "RE", "message": f"Non-zero exit code.\n{err_log}", "actual_output": ""}
        elif result["status"] == "System Error":
            return {"verdict": "SE", "message": result.get("message", "Internal Docker Error"), "actual_output": ""}

        # Pass work_dir so expected.txt can be downloaded safely into the sandbox
        verdict = evaluate_output(work_dir, output_txt_path, expected_output, expected_s3_key)
        actual_output_snippet = read_file_safely(output_txt_path)

        return {
            "verdict": verdict,
            "actual_output": actual_output_snippet
        }
    except Exception as e:
        return {"verdict": "SE", "message": str(e), "actual_output": ""}
    finally:
        shutil.rmtree(work_dir, ignore_errors=True)