import docker
import os
import shutil
import itertools
import math
from typing import Optional

# Import the shared S3 fetcher from our centralized config
from config import fetch_from_s3

# --- CONFIGURATION ---
client = docker.from_env()
SANDBOX_BASE = "/sandbox_shared"
CACHE_BASE = os.path.join(SANDBOX_BASE, "cache")

# --- CACHING & I/O HELPERS ---
def ensure_cached_testcase(work_dir, problem_id, tc_id, input_data, expected_output, input_s3_key, expected_s3_key):
    if str(problem_id) == "custom":
        tc_dir = work_dir 
    else:
        tc_dir = os.path.join(CACHE_BASE, str(problem_id), str(tc_id))
        os.makedirs(tc_dir, exist_ok=True)
        
    input_path = os.path.join(tc_dir, 'input.txt')
    expected_path = os.path.join(tc_dir, 'expected.txt')
    
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
    if language == 'python':
        with open(os.path.join(work_dir, 'solution.py'), 'w', encoding='utf-8') as f: f.write(source_code)
        return None 

    container = None
    try:
        if language == 'cpp':
            with open(os.path.join(work_dir, 'solution.cpp'), 'w', encoding='utf-8') as f: f.write(source_code)
            cmd = 'sh -c "g++ -O2 -w solution.cpp -o solution.out > compile.log 2>&1"'
            img = "campus-cpp"
        elif language == 'java':
            with open(os.path.join(work_dir, 'Main.java'), 'w', encoding='utf-8') as f: f.write(source_code)
            cmd = 'sh -c "javac Main.java > compile.log 2>&1"'
            img = "campus-java"
        else:
            return {"verdict": "CE", "message": "Unsupported language"}

        compilation_mem = "1g" if language == 'java' else "512m"

        container = client.containers.run(
            image=img, command=cmd, 
            volumes={'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'rw'}},
            working_dir=work_dir, detach=True, network_disabled=True, 
            mem_limit=compilation_mem, memswap_limit=compilation_mem, auto_remove=True
        )
        
        res = container.wait(timeout=10)
        if res['StatusCode'] != 0:
            logs = read_file_safely(os.path.join(work_dir, 'compile.log'))
            return {"verdict": "CE", "message": logs.strip()}
        return None

    except Exception as e:
        if container:
            try: container.kill()
            except: pass
        print(f"[!] Compilation Container Error: {e}")
        return {"verdict": "CE", "message": "Compilation Timed Out or Failed"}

def run_all_cases(language: str, work_dir: str, tc_meta: list, time_limit_seconds: float, memory_limit_kb: int):
    volumes = {'sandbox_volume': {'bind': SANDBOX_BASE, 'mode': 'rw'}}
    container = None
    
    script_path = os.path.join(work_dir, 'execute.sh')
    
    timeout_int = math.ceil(time_limit_seconds)
    
    if language == 'python':
        img = "campus-python"
    elif language == 'cpp':
        img = "campus-cpp"
    elif language == 'java':
        img = "campus-java"
        
    with open(script_path, 'w', encoding='utf-8') as f:
        f.write("#!/bin/sh\n")
        for meta in tc_meta:
            idx = meta['index']
            input_path = meta['input_path']
            
            if language == 'python':
                cmd = f"python solution.py < {input_path} > out_{idx}.txt 2> err_{idx}.txt"
            elif language == 'cpp':
                cmd = f"./solution.out < {input_path} > out_{idx}.txt 2> err_{idx}.txt"
            elif language == 'java':
                cmd = f"java -Xmx{int(memory_limit_kb/1024)}m Main < {input_path} > out_{idx}.txt 2> err_{idx}.txt"
                
            f.write(f"timeout {timeout_int} sh -c '{cmd}'\n")
            f.write(f"RES=$?\n")
            f.write(f"echo $RES > status_{idx}.txt\n")
            f.write(f"if [ $RES -ne 0 ]; then exit 0; fi\n")

    try:
        container = client.containers.run(
            image=img, command="sh execute.sh", volumes=volumes, working_dir=work_dir, 
            detach=True, network_disabled=True, 
            mem_limit=f"{memory_limit_kb}k", memswap_limit=f"{memory_limit_kb}k", 
            auto_remove=True, cpu_period=100000, cpu_quota=100000, pids_limit=64, 
            cap_drop=["ALL"], security_opt=["no-new-privileges"]
        )
        
        max_wait = (time_limit_seconds * len(tc_meta)) + 5.0
        container.wait(timeout=max_wait)
        return {"status": "Success"}
        
    except Exception as e:
        if container:
            try: container.kill() 
            except: pass
        if "Timeout" in str(e) or "Read timed out" in str(e): 
            return {"status": "Container Timeout Exceeded"}
        
        print(f"[!] Container Execution Error: {e}")
        return {"status": "System Error", "message": "Internal execution environment failed."}

def evaluate_output(actual_output_file_path: str, cached_expected_path: str) -> str:
    def get_clean_lines(path):
        if not os.path.exists(path):
            return
            
        with open(path, 'r', encoding='utf-8', errors='replace') as f:
            lines = f.readlines()
            
        while lines and lines[-1].strip() == "":
            lines.pop()
            
        for line in lines:
            yield line.rstrip()

    actual_iter = get_clean_lines(actual_output_file_path)
    expected_iter = get_clean_lines(cached_expected_path)

    sentinel = object()
    for a_line, e_line in itertools.zip_longest(actual_iter, expected_iter, fillvalue=sentinel):
        if a_line is sentinel or e_line is sentinel:
            return "WA"
            
        if a_line != e_line:
            return "WA"
            
    return "AC"

# --- THE MASTER GRADER ---
def grade_submission(submission_id: str, problem_id: str, language: str, source_code: Optional[str], source_s3_key: Optional[str], test_cases: list, time_limit_ms: int, memory_limit_kb: int):
    time_limit_sec = time_limit_ms / 1000.0
    work_dir = os.path.join(SANDBOX_BASE, str(submission_id))
    os.makedirs(work_dir, exist_ok=True)
    os.chmod(SANDBOX_BASE, 0o755)
    
    os.chmod(work_dir, 0o777)
    try:
        os.chown(work_dir, 1001, 1001) 
    except PermissionError:
        print(f"[!] Warning: Could not chown {work_dir}. Running as non-root host?")
        pass
    
    try:
        if source_s3_key:
            file_ext = "cpp" if language == "cpp" else "py" if language == "python" else "java"
            download_path = os.path.join(work_dir, f"source.{file_ext}")
            
            try:
                fetch_from_s3(source_s3_key, download_path)
                with open(download_path, 'r', encoding='utf-8') as f:
                    source_code = f.read()
            except Exception as e:
                print(f"[!] Failed to fetch source from S3: {e}")
                return {"verdict": "SE", "message": "Failed to retrieve source code for execution.", "actual_output": ""}

        if not source_code:
            return {"verdict": "SE", "message": "Source code is empty.", "actual_output": ""}

        compile_err = compile_code(language, work_dir, source_code)
        if compile_err: return compile_err 
            
        tc_meta = []
        for idx, tc in enumerate(test_cases):
            tc_id = tc.get('test_case_id', f"custom_{idx}")
            is_hidden = tc.get('is_hidden', False) 
            
            input_path, expected_path = ensure_cached_testcase(
                work_dir, problem_id, tc_id, tc.get('input_data'), tc.get('expected_output'), 
                tc.get('input_s3_key'), tc.get('expected_s3_key')
            )
            tc_meta.append({
                'index': idx,
                'input_path': input_path,
                'expected_path': expected_path,
                'is_hidden': is_hidden
            })
            
        run_result = run_all_cases(language, work_dir, tc_meta, time_limit_sec, memory_limit_kb)
        
        if run_result["status"] == "Container Timeout Exceeded":
            return {"verdict": "SE", "message": "Container critically timed out.", "actual_output": ""}
        elif run_result["status"] != "Success":
            return {"verdict": "SE", "message": run_result.get("message", "System Error"), "actual_output": ""}

        for meta in tc_meta:
            idx = meta['index']
            expected_path = meta['expected_path']
            is_hidden = meta['is_hidden']
            
            status_file = os.path.join(work_dir, f'status_{idx}.txt')
            out_file = os.path.join(work_dir, f'out_{idx}.txt')
            err_file = os.path.join(work_dir, f'err_{idx}.txt')
            
            if not os.path.exists(status_file):
                return {"verdict": "SE", "message": f"Execution halted unexpectedly before Test Case {idx+1}", "actual_output": ""}

            status_code_str = read_file_safely(status_file).strip()
            if not status_code_str:
                return {"verdict": "SE", "message": f"Empty status code for Test Case {idx+1}", "actual_output": ""}
                
            status_code = int(status_code_str)
            
            if status_code in [124, 137, 143]: 
                return {"verdict": "TLE", "message": f"Time Limit Exceeded on Test Case {idx+1}", "actual_output": ""}
            elif status_code != 0:
                if str(problem_id) == "custom":
                    err_log = read_file_safely(err_file)
                    return {"verdict": "RE", "message": f"Runtime Error on Test Case {idx+1}.\n{err_log}", "actual_output": ""}
                else:
                    return {"verdict": "RE", "message": f"Runtime Error on Test Case {idx+1}. (Stack trace hidden)", "actual_output": ""}
                
            verdict = evaluate_output(out_file, expected_path)
            if verdict != "AC":
                if str(problem_id) == "custom" or not is_hidden:
                    actual_out = read_file_safely(out_file, max_chars=1000)
                    expected_out = read_file_safely(expected_path, max_chars=1000)
                    msg = f"Wrong Answer on Test Case {idx+1}.\n\nExpected Output:\n{expected_out}\n\nYour Output:\n{actual_out}"
                    return {"verdict": "WA", "message": msg.strip(), "actual_output": actual_out}
                else:
                    return {"verdict": "WA", "message": f"Wrong Answer on Hidden Test Case {idx+1}", "actual_output": ""}
                
        last_idx = tc_meta[-1]['index']
        return {"verdict": "AC", "actual_output": read_file_safely(os.path.join(work_dir, f'out_{last_idx}.txt'))}
        
    except Exception as e:
        print(f"[!] Master Grader Exception (Problem {problem_id}): {e}")
        return {"verdict": "SE", "message": "System Error: The execution engine encountered an unexpected failure.", "actual_output": ""}
    finally:
        shutil.rmtree(work_dir, ignore_errors=True)