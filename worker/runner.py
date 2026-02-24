import docker
import os
import tempfile

# Initialize the Docker client
client = docker.from_env()

#Python
def execute_python_code(source_code: str, input_data: str, time_limit_seconds: float = 2):
    """
    Executes Python code inside a secure Docker container using file-based I/O.
    """
    with tempfile.TemporaryDirectory() as temp_dir:
        # 1. Write the user's code to a file
        source_path = os.path.join(temp_dir, 'solution.py')
        with open(source_path, 'w', encoding='utf-8') as f:
            f.write(source_code)
            
        # 2. Write the test case input to a file
        input_path = os.path.join(temp_dir, 'input.txt')
        with open(input_path, 'w', encoding='utf-8') as f:
            f.write(input_data)

        try:
            # 3. Spin up the container
            container = client.containers.run(
                image="campus-python",
                command='sh -c "python /sandbox/solution.py < /sandbox/input.txt"',
                volumes={temp_dir: {'bind': '/sandbox', 'mode': 'ro'}},
                working_dir='/sandbox',
                detach=True,           
                network_disabled=True, 
                mem_limit="256m",      
                cpu_period=100000,
                cpu_quota=100000,
                
                # --- SECURITY PATCHES ---
                pids_limit=64,                     # Prevents Fork Bombs
                cap_drop=["ALL"],                  # Strips root capabilities
                security_opt=["no-new-privileges"] # Prevents privilege escalation
            )

            # 4. Wait for the container to finish, enforcing the Time Limit (TLE)
            result = container.wait(timeout=time_limit_seconds)
            exit_code = result['StatusCode']

            # 5. Grab the output (stdout and stderr) 
            # SECURITY PATCH: tail=500 prevents Output Flooding (OOM crashes)
            logs = container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')

            # Clean up the container
            container.remove(force=True)

            return {
                "status": "Success" if exit_code == 0 else "Runtime Error",
                "exit_code": exit_code,
                "output": logs.strip()
            }

        except docker.errors.ContainerError as e:
            return {"status": "Runtime Error", "output": str(e)}
        except Exception as e:
            # Catching Time Limit Exceeded
            try:
                container.remove(force=True)
            except:
                pass
            if "Timeout" in str(e) or "Read timed out" in str(e):
                return {"status": "Time Limit Exceeded", "output": ""}
            return {"status": "System Error", "output": str(e)}

#CPP
def execute_cpp_code(source_code: str, input_data: str, time_limit_seconds: float = 2):
    """
    Executes C++ code inside a secure Docker container using a compile-then-run pipeline.
    """
    with tempfile.TemporaryDirectory() as temp_dir:
        # 1. Write the user's C++ code to a file
        source_path = os.path.join(temp_dir, 'solution.cpp')
        with open(source_path, 'w', encoding='utf-8') as f:
            f.write(source_code)
            
        # 2. Write the test case input
        input_path = os.path.join(temp_dir, 'input.txt')
        with open(input_path, 'w', encoding='utf-8') as f:
            f.write(input_data)

        # --- STEP 1: COMPILATION ---
        try:
            compile_container = client.containers.run(
                image="campus-cpp",
                command='g++ -O2 -w /sandbox/solution.cpp -o /sandbox/solution.out',
                volumes={temp_dir: {'bind': '/sandbox', 'mode': 'rw'}},
                working_dir='/sandbox',
                detach=True,
                network_disabled=True,
                mem_limit="512m",
                
                # --- SECURITY PATCHES ---
                pids_limit=64,
                cap_drop=["ALL"],
                security_opt=["no-new-privileges"]
            )
            
            # Wait for compilation to finish
            compile_result = compile_container.wait(timeout=10)
            
            # If exit code is not 0, it's a Compilation Error
            if compile_result['StatusCode'] != 0:
                # SECURITY PATCH: tail=500
                logs = compile_container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
                compile_container.remove(force=True)
                return {"status": "Compilation Error", "output": logs.strip()}
                
            compile_container.remove(force=True)
            
        except Exception as e:
            return {"status": "System Error", "output": f"Compilation failed: {str(e)}"}

        # --- STEP 2: EXECUTION ---
        try:
            run_container = client.containers.run(
                image="campus-cpp",
                command='sh -c "/sandbox/solution.out < /sandbox/input.txt"',
                volumes={temp_dir: {'bind': '/sandbox', 'mode': 'ro'}},
                working_dir='/sandbox',
                detach=True,           
                network_disabled=True, 
                mem_limit="256m",      
                cpu_period=100000,
                cpu_quota=100000,
                
                # --- SECURITY PATCHES ---
                pids_limit=64,
                cap_drop=["ALL"],
                security_opt=["no-new-privileges"]
            )

            result = run_container.wait(timeout=time_limit_seconds)
            exit_code = result['StatusCode']
            # SECURITY PATCH: tail=500
            logs = run_container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
            run_container.remove(force=True)

            return {
                "status": "Success" if exit_code == 0 else "Runtime Error",
                "exit_code": exit_code,
                "output": logs.strip()
            }

        except Exception as e:
            try:
                run_container.remove(force=True)
            except:
                pass
            if "Timeout" in str(e) or "Read timed out" in str(e):
                return {"status": "Time Limit Exceeded", "output": ""}
            return {"status": "System Error", "output": str(e)}
        

#Java
def execute_java_code(source_code: str, input_data: str, time_limit_seconds: float = 2):
    """
    Executes Java code inside a secure Docker container using a compile-then-run pipeline.
    """
    with tempfile.TemporaryDirectory() as temp_dir:
        # 1. Write the user's Java code to Main.java
        source_path = os.path.join(temp_dir, 'Main.java')
        with open(source_path, 'w', encoding='utf-8') as f:
            f.write(source_code)
            
        # 2. Write the test case input
        input_path = os.path.join(temp_dir, 'input.txt')
        with open(input_path, 'w', encoding='utf-8') as f:
            f.write(input_data)

        # --- STEP 1: COMPILATION ---
        try:
            compile_container = client.containers.run(
                image="campus-java",
                command='javac /sandbox/Main.java',
                volumes={temp_dir: {'bind': '/sandbox', 'mode': 'rw'}},
                working_dir='/sandbox',
                detach=True,
                network_disabled=True,
                mem_limit="512m",
                
                # --- SECURITY PATCHES ---
                pids_limit=64,
                cap_drop=["ALL"],
                security_opt=["no-new-privileges"]
            )
            
            compile_result = compile_container.wait(timeout=10)
            
            if compile_result['StatusCode'] != 0:
                # SECURITY PATCH: tail=500
                logs = compile_container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
                compile_container.remove(force=True)
                return {"status": "Compilation Error", "output": logs.strip()}
                
            compile_container.remove(force=True)
            
        except Exception as e:
            return {"status": "System Error", "output": f"Compilation failed: {str(e)}"}

        # --- STEP 2: EXECUTION ---
        try:
            run_container = client.containers.run(
                image="campus-java",
                command='sh -c "java -Xmx256m Main < /sandbox/input.txt"',
                volumes={temp_dir: {'bind': '/sandbox', 'mode': 'ro'}},
                working_dir='/sandbox',
                detach=True,           
                network_disabled=True, 
                mem_limit="512m",      
                cpu_period=100000,
                cpu_quota=100000,
                
                # --- SECURITY PATCHES ---
                pids_limit=64,
                cap_drop=["ALL"],
                security_opt=["no-new-privileges"]
            )

            result = run_container.wait(timeout=time_limit_seconds)
            exit_code = result['StatusCode']
            # SECURITY PATCH: tail=500
            logs = run_container.logs(stdout=True, stderr=True, tail=500).decode('utf-8')
            run_container.remove(force=True)

            return {
                "status": "Success" if exit_code == 0 else "Runtime Error",
                "exit_code": exit_code,
                "output": logs.strip()
            }

        except Exception as e:
            try:
                run_container.remove(force=True)
            except:
                pass
            if "Timeout" in str(e) or "Read timed out" in str(e):
                return {"status": "Time Limit Exceeded", "output": ""}
            return {"status": "System Error", "output": str(e)}

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