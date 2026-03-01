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

        cursor.execute("""
            SELECT s.source_code, s.language, s.problem_id, 
                   p.time_limit_ms, p.memory_limit_kb 
            FROM submissions s
            JOIN problems p ON s.problem_id = p.problem_id
            WHERE s.submission_id = %s
        """, (submission_id,))
        submission = cursor.fetchone()

        if not submission:
            print(f"[!] Submission {submission_id} not found in database.")
            return

        # 💡 UPDATED: Fetching S3 keys instead of local Polygon paths
        cursor.execute("""
            SELECT input_data, expected_output, input_s3_key, expected_s3_key
            FROM test_cases 
            WHERE problem_id = %s
        """, (submission.get('problem_id'),))
        test_cases = cursor.fetchall()

        if len(test_cases) == 0:
            cursor.execute("UPDATE submissions SET status = 'SE - No Test Cases' WHERE submission_id = %s", (submission_id,))
            conn.commit()
            print(f"[!] System Error: No test cases found for Problem {submission.get('problem_id')}")
            return

        final_verdict = 'AC'
        final_message = 'All test cases passed! 🏆'
        
        for idx, tc in enumerate(test_cases):
            print(f"[-] Running Test Case {idx + 1}/{len(test_cases)}...")
            
            # 💡 UPDATED: Passing memory limits and S3 keys to the runner
            result = grade_submission(
                language=submission.get('language'),
                source_code=submission.get('source_code'),
                input_data=tc.get('input_data'),
                expected_output=tc.get('expected_output'),
                time_limit_ms=submission.get('time_limit_ms'),
                memory_limit_kb=submission.get('memory_limit_kb'),
                input_s3_key=tc.get('input_s3_key'),
                expected_s3_key=tc.get('expected_s3_key')
            )
            
            if result['verdict'] != 'AC':
                final_verdict = result['verdict']
                if final_verdict == 'WA':
                    final_message = f"Wrong Answer on Test Case {idx + 1}.\nTest data is hidden to prevent hardcoding."
                else:
                    final_message = result.get('message', f"Verdict: {final_verdict} on Test Case {idx + 1}")
                break

        cursor.execute(
            """
            UPDATE submissions 
            SET status = %s, error_logs = %s 
            WHERE submission_id = %s
            """,
            (final_verdict, final_message, submission_id)
        )
        conn.commit()
        print(f"[+] Submission {submission_id} completed. Final Verdict: {final_verdict}\n")

    except Exception as e:
        print(f"[!] Database/Execution Error: {str(e)}")
        conn.rollback()
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
                result = grade_submission(
                    language=submission_data.get('language'),
                    source_code=submission_data.get('source_code'),
                    input_data=submission_data.get('custom_input', ''),
                    expected_output="", 
                    time_limit_ms=2000
                )
                output_to_show = result.get('actual_output')
                if result['verdict'] in ['CE', 'RE', 'TLE', 'SE']:
                    output_to_show = result.get('message', f"Error: {result['verdict']}")
                redis_client.set(f"run_result:{run_id}", json.dumps({
                    "status": "Completed", "output": output_to_show, "verdict": result['verdict']
                }), ex=600) 
            else:
                sub_id = submission_data.get('submission_id')
                if sub_id:
                    print(f"\n[+] Picked up submission ID: {sub_id}")
                    process_submission(sub_id)

if __name__ == "__main__":
    start_worker()