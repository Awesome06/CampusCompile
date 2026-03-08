import redis
import json
import psycopg2
import os
from psycopg2.extras import RealDictCursor
from runner import grade_submission

# --- CONFIGURATION ---
DB_CONFIG = {
    "dbname": "CampusCompile_db",
    "user": "campus_app",            
    "password": "app", 
    "host": os.getenv("DB_HOST", "localhost"),
    "port": "5432"
}

redis_client = redis.Redis(host=os.getenv("REDIS_HOST", "redis"), port=6379, db=0, decode_responses=True)
QUEUE_NAME = 'submission_queue'

def get_db_connection():
    return psycopg2.connect(**DB_CONFIG, cursor_factory=RealDictCursor)

def process_submission(submission_id):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    try:
        cursor.execute("UPDATE submissions SET status = 'Running' WHERE submission_id = %s", (submission_id,))
        conn.commit()

        redis_client.publish(f"submission_updates:{submission_id}", json.dumps({
            "status": "Running", "message": "Compiling and executing..."
        }))

        cursor.execute("""
            SELECT s.source_code_s3_key, s.language, s.problem_id, 
                   p.time_limit_ms, p.memory_limit_kb 
            FROM submissions s
            JOIN problems p ON s.problem_id = p.problem_id
            WHERE s.submission_id = %s
        """, (submission_id,))
        submission = cursor.fetchone()

        if not submission:
            print(f"[!] Submission {submission_id} not found in database.")
            return

        cursor.execute("""
            SELECT test_case_id, input_data, expected_output, input_s3_key, expected_s3_key
            FROM test_cases 
            WHERE problem_id = %s
        """, (submission.get('problem_id'),))
        test_cases = cursor.fetchall()

        if len(test_cases) == 0:
            cursor.execute("UPDATE submissions SET status = 'SE', error_logs = 'No Test Cases' WHERE submission_id = %s", (submission_id,))
            conn.commit()
            print(f"[!] System Error: No test cases found for Problem {submission.get('problem_id')}")
            return

        print(f"[*] Grading Submission {submission_id} across {len(test_cases)} test cases...")
        
        # 👇 CHANGED: Pass None for source_code, pass the S3 key instead
        result = grade_submission(
            submission_id=submission_id,
            problem_id=submission.get('problem_id'),
            language=submission.get('language'),
            source_code=None, 
            source_s3_key=submission.get('source_code_s3_key'), # 👈 NEW PARAMETER
            test_cases=test_cases,
            time_limit_ms=submission.get('time_limit_ms', 2000),
            memory_limit_kb=submission.get('memory_limit_kb', 256000)
        )

        final_verdict = result['verdict']
        final_message = result.get('message', 'All test cases passed! 🏆') if final_verdict == 'AC' else result.get('message', f'Verdict: {final_verdict}')

        cursor.execute(
            "UPDATE submissions SET status = %s, error_logs = %s WHERE submission_id = %s",
            (final_verdict, final_message, submission_id)
        )
        conn.commit()
        
        # 👇 2. PUBLISH Final Verdict to Redis
        redis_client.publish(f"submission_updates:{submission_id}", json.dumps({
            "status": final_verdict, "message": final_message
        }))
        
        print(f"[+] Submission {submission_id} completed. Final Verdict: {final_verdict}\n")

    except Exception as e:
        print(f"[!] Database/Execution Error: {str(e)}")
        conn.rollback()
        # Publish error so frontend doesn't hang
        redis_client.publish(f"submission_updates:{submission_id}", json.dumps({
            "status": "SE", "message": "System Error during execution"
        }))
    finally:
        cursor.close()
        conn.close()
        
def start_worker():
    print(f"[*] Worker started. Listening to Redis queue: '{QUEUE_NAME}'...")
    while True:
        queue, message = redis_client.brpop(QUEUE_NAME)
        if message:
            submission_data = json.loads(message)
            
            if submission_data.get('is_custom'):
                run_id = submission_data.get('run_id')
                print(f"\n[+] Processing Custom Run: {run_id}")

                redis_client.publish(f"run_updates:{run_id}", json.dumps({"status": "Running"}))
                
                # Format custom run to match the batch runner signature
                custom_tc = [{
                    "test_case_id": "custom",
                    "input_data": submission_data.get('custom_input', ''),
                    "expected_output": ""
                }]
                
                # 👇 CHANGED: Added source_s3_key=None to match the updated signature
                result = grade_submission(
                    submission_id=run_id,
                    problem_id="custom",
                    language=submission_data.get('language'),
                    source_code=submission_data.get('source_code'),
                    source_s3_key=None, # 👈 THE FIX
                    test_cases=custom_tc,
                    time_limit_ms=2000,
                    memory_limit_kb=256000
                )
                
                output_to_show = result.get('actual_output')
                if result['verdict'] in ['CE', 'RE', 'TLE', 'SE']:
                    output_to_show = result.get('message', f"Error: {result['verdict']}")
                    
                redis_client.set(f"run_result:{run_id}", json.dumps({
                    "status": "Completed", "output": output_to_show, "verdict": result['verdict']
                }), ex=600) 

                redis_client.publish(f"run_updates:{run_id}", json.dumps({
                    "status": "Completed", "output": output_to_show, "verdict": result['verdict']
                }))
            else:
                sub_id = submission_data.get('submission_id')
                if sub_id:
                    print(f"\n[+] Picked up submission ID: {sub_id}")
                    process_submission(sub_id)

if __name__ == "__main__":
    start_worker()