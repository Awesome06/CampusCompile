import redis
import json
import psycopg2
from psycopg2.extras import RealDictCursor
from runner import grade_submission

# --- CONFIGURATION ---
# Match these to the PostgreSQL credentials you set up in Phase 1
DB_CONFIG = {
    "dbname": "CampusCompile_db",
    "user": "campus_app",            # The app user we created
    "password": "app", # The password you set for campus_app
    "host": "localhost",
    "port": "5432"
}

redis_client = redis.Redis(host='localhost', port=6379, db=0, decode_responses=True)
QUEUE_NAME = 'submission_queue'

def get_db_connection():
    return psycopg2.connect(**DB_CONFIG, cursor_factory=RealDictCursor)

def process_submission(submission_id):
    conn = get_db_connection()
    cursor = conn.cursor()
    
    try:
        # 1. Update status to 'Running' so the user interface knows it started
        cursor.execute("UPDATE submissions SET status = 'Running' WHERE submission_id = %s", (submission_id,))
        conn.commit()

        # 2. Fetch the submission details and problem constraints
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

        # 3. Fetch all test cases for this problem
        cursor.execute("""
            SELECT input_data, expected_output 
            FROM test_cases 
            WHERE problem_id = %s
        """, (submission.get('problem_id'),))
        test_cases = cursor.fetchall()

        if len(test_cases) == 0:
            cursor.execute("UPDATE submissions SET status = 'SE - No Test Cases' WHERE submission_id = %s", (submission_id,))
            conn.commit()
            print(f"[!] System Error: No test cases found for Problem {submission.get('problem_id')}")
            return

        # 4. Loop through test cases and grade
        final_verdict = 'AC'
        max_time_ms = 0
        
        for idx, tc in enumerate(test_cases):
            print(f"[-] Running Test Case {idx + 1}/{len(test_cases)}...")
            
            result = grade_submission(
                language=submission.get('language'),
                source_code=submission.get('source_code'),
                input_data=tc.get('input_data'),
                expected_output=tc.get('expected_output'),
                time_limit_ms=submission.get('time_limit_ms')
            )
            
            # If a test case fails (WA, TLE, CE, RE, SE), we break early! 
            if result['verdict'] != 'AC':
                final_verdict = result['verdict']
                break

        # 5. Save the final verdict back to the database
        cursor.execute("""
            UPDATE submissions 
            SET status = %s 
            WHERE submission_id = %s
        """, (final_verdict, submission_id))
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
            sub_id = submission_data.get('submission_id')
            print(f"\n[+] Picked up submission ID: {sub_id}")
            process_submission(sub_id)

if __name__ == "__main__":
    start_worker()