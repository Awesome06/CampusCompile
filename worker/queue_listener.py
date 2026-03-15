import redis
import sys
import json
import psycopg2
import os
import traceback
from datetime import timezone
from psycopg2.extras import RealDictCursor
from runner import grade_submission
from moss_auditor import run_moss_audit

# --- STRICT ENVIRONMENT VALIDATION ---
def get_required_env(var_name: str) -> str:
    value = os.getenv(var_name)
    if not value:
        # Write cleanly to stderr and terminate without a messy stack trace
        sys.stderr.write(f"FATAL STARTUP ERROR: Required environment variable '{var_name}' is missing or empty.\n")
        sys.exit(1)
    return value

# --- CONFIGURATION ---
DB_CONFIG = {
    "dbname": "CampusCompile_db",
    "user": get_required_env("POSTGRES_USER"),         # <-- Wired up
    "password": get_required_env("POSTGRES_PASSWORD"), # <-- Wired up
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

        # 👇 FIX: Calculate 'elapsed_minutes' safely inside PostgreSQL
        cursor.execute("""
            SELECT s.source_code_s3_key, s.language, s.problem_id, s.user_id, s.contest_id, s.submitted_at,
                   p.time_limit_ms, p.memory_limit_kb,
                   c.start_time as contest_start,
                   EXTRACT(EPOCH FROM (s.submitted_at - c.start_time))/60.0 as elapsed_minutes
            FROM submissions s
            JOIN problems p ON s.problem_id = p.problem_id
            LEFT JOIN contests c ON s.contest_id = c.contest_id
            WHERE s.submission_id = %s
        """, (submission_id,))
        submission = cursor.fetchone()

        if not submission:
            print(f"[!] Submission {submission_id} not found in database.")
            return

        cursor.execute("""
            SELECT test_case_id, input_s3_key, expected_s3_key
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
        
        result = grade_submission(
            submission_id=submission_id,
            problem_id=submission.get('problem_id'),
            language=submission.get('language'),
            source_code=None, 
            source_s3_key=submission.get('source_code_s3_key'),
            test_cases=test_cases,
            time_limit_ms=submission.get('time_limit_ms', 2000),
            memory_limit_kb=submission.get('memory_limit_kb', 256000)
        )

        final_verdict = result['verdict']
        final_message = result.get('message', 'All test cases passed! 🏆') if final_verdict == 'AC' else result.get('message', f'Verdict: {final_verdict}')

        # 1. Save the verdict to PostgreSQL
        cursor.execute(
            "UPDATE submissions SET status = %s, error_logs = %s WHERE submission_id = %s",
            (final_verdict, final_message, submission_id)
        )
        conn.commit()
        
        # 2. Phase 3 ICPC Leaderboard Engine
        if final_verdict == 'AC' and submission.get('contest_id'):
            contest_id = submission['contest_id']
            user_id = submission['user_id']
            problem_id = submission['problem_id']
            submitted_at = submission['submitted_at']

            cursor.execute("""
                SELECT COUNT(*) as solved 
                FROM submissions 
                WHERE user_id = %s AND problem_id = %s AND contest_id = %s 
                  AND status = 'AC' AND submission_id != %s
            """, (user_id, problem_id, contest_id, submission_id))
            
            if cursor.fetchone()['solved'] == 0:
                cursor.execute("""
                    SELECT COUNT(*) as fails 
                    FROM submissions 
                    WHERE user_id = %s AND problem_id = %s AND contest_id = %s 
                      AND status IN ('WA', 'TLE', 'RE', 'MLE', 'SE', 'CE') 
                      AND submitted_at < %s
                """, (user_id, problem_id, contest_id, submitted_at))
                fails = cursor.fetchone()['fails']

                # 👇 FIX: Extract pre-calculated safe float from DB query
                elapsed_minutes = max(0, float(submission.get('elapsed_minutes') or 0))
                penalty_minutes = elapsed_minutes + (fails * 20)

                score_increment = 1.0 - (penalty_minutes / 100000.0)

                redis_client.zincrby(f"contest:leaderboard:{contest_id}", score_increment, user_id)
                redis_client.sadd("dirty_contests", contest_id)
                print(f"[+] ICPC Score Updated for {user_id}. (+1 Solve, {penalty_minutes:.2f} Penalty Mins)")

        # 3. Publish Final Verdict to Redis
        redis_client.publish(f"submission_updates:{submission_id}", json.dumps({
            "status": final_verdict, "message": final_message
        }))
        
        print(f"[+] Submission {submission_id} completed. Final Verdict: {final_verdict}\n")

    except Exception as e:
        # 👇 FIX: Aggressive error logging directly to your terminal
        print(f"\n[!] CRITICAL PYTHON CRASH in queue_listener.py:")
        traceback.print_exc() 
        conn.rollback()
        
        # We pass the EXACT error text to the frontend console instead of a generic message
        redis_client.publish(f"submission_updates:{submission_id}", json.dumps({
            "status": "SE", "message": f"Worker Crash: {str(e)}"
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
            
            if submission_data.get('job_type') == 'moss_audit':
                contest_id = submission_data.get('contest_id')
                problem_id = submission_data.get('problem_id')
                print(f"\n[+] Picked up MOSS Audit Job for Problem: {problem_id}")
                run_moss_audit(contest_id)
                
            elif submission_data.get('is_custom'):
                run_id = submission_data.get('run_id')
                print(f"\n[+] Processing Custom Run: {run_id}")

                redis_client.publish(f"run_updates:{run_id}", json.dumps({"status": "Running"}))
                
                custom_tc = [{
                    "test_case_id": "custom",
                    "input_data": submission_data.get('custom_input', ''),
                    "expected_output": ""
                }]
                
                result = grade_submission(
                    submission_id=run_id,
                    problem_id="custom",
                    language=submission_data.get('language'),
                    source_code=submission_data.get('source_code'),
                    source_s3_key=None, 
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