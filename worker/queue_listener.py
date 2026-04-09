import json
import traceback
import sys
import threading
import time # NEW: For Redis backoff
import logging
from concurrent.futures import ThreadPoolExecutor
from runner import grade_submission
from moss_auditor import run_moss_audit
import redis

logger = logging.getLogger('queue_listener')

# Import our robust, thread-safe configuration elements
from config import (
    redis_client, 
    get_db_connection, 
    release_db_connection, 
    ConfigurationError, 
    get_s3_client
)

# --- CONFIGURATION ---
QUEUE_NAME = 'submission_queue'
MAX_WORKERS = 10 

# Semaphore to apply backpressure: prevents unbounded memory growth if Redis floods us
job_semaphore = threading.Semaphore(MAX_WORKERS) 

def process_submission(submission_id):
    """Handles the full lifecycle of grading a single submission."""
    conn = None
    cursor = None
    
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        
        cursor.execute("UPDATE submissions SET status = 'Running' WHERE submission_id = %s", (submission_id,))
        conn.commit()

        redis_client.publish(f"submission_updates:{submission_id}", json.dumps({
            "status": "Running", "message": "Compiling and executing..."
        }))

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
            logger.error(f"[!] Submission {submission_id} not found in database.")
            return

        cursor.execute("""
            SELECT test_case_id, input_s3_key, expected_s3_key, is_hidden
            FROM test_cases 
            WHERE problem_id = %s
            ORDER BY input_s3_key ASC 
        """, (submission.get('problem_id'),))
        test_cases = cursor.fetchall()

        if len(test_cases) == 0:
            cursor.execute("UPDATE submissions SET status = 'SE', error_logs = 'No Test Cases' WHERE submission_id = %s", (submission_id,))
            conn.commit()
            logger.error(f"[!] System Error: No test cases found for Problem {submission.get('problem_id')}")
            return

        logger.info(f"[*] Grading Submission {submission_id} across {len(test_cases)} test cases...")

        lang = submission.get('language')
        base_time_ms = submission.get('time_limit_ms', 2000)
        base_mem_kb = submission.get('memory_limit_kb', 256000)

        limits = {'time': 1.0, 'memory': 1.0}

        actual_time_ms = int(base_time_ms * limits['time'])
        actual_mem_kb = int(base_mem_kb * limits['memory'])
        
        result = grade_submission(
            submission_id=submission_id,
            problem_id=submission.get('problem_id'),
            language=lang,
            source_code=None, 
            source_s3_key=submission.get('source_code_s3_key'),
            test_cases=test_cases,
            time_limit_ms=actual_time_ms,
            memory_limit_kb=actual_mem_kb
        )

        final_verdict = result['verdict']
        final_message = result.get('message', 'All test cases passed! 🚀') if final_verdict == 'AC' else result.get('message', f'Verdict: {final_verdict}')

        cursor.execute(
            "UPDATE submissions SET status = %s, error_logs = %s WHERE submission_id = %s",
            (final_verdict, final_message, submission_id)
        )
        conn.commit()
        
        # ICPC Leaderboard Engine Updates
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

                elapsed_minutes = max(0, float(submission.get('elapsed_minutes') or 0))
                penalty_minutes = elapsed_minutes + (fails * 20)

                score_increment = 1.0 - (penalty_minutes / 100000.0)

                redis_client.zincrby(f"contest:leaderboard:{contest_id}", score_increment, user_id)
                redis_client.sadd("dirty_contests", contest_id)
                logger.info(f"[+] ICPC Score Updated for {user_id}. (+1 Solve, {penalty_minutes:.2f} Penalty Mins)")

        redis_client.publish(f"submission_updates:{submission_id}", json.dumps({
            "status": final_verdict, "message": final_message
        }))
        
        logger.info(f"[+] Submission {submission_id} completed. Final Verdict: {final_verdict}\n")

    except ConfigurationError as ce:
        logger.error(f"\n[!] INFRASTRUCTURE ERROR: {ce}")
        if conn and cursor:
            try:
                conn.rollback()
                cursor.execute("UPDATE submissions SET status = 'SE', error_logs = 'System Error: Infrastructure temporarily unavailable.' WHERE submission_id = %s", (submission_id,))
                conn.commit()
            except Exception as db_err:
                logger.error(f"[!] Failed to log SE to DB: {db_err}")
                
        try:
            redis_client.publish(f"submission_updates:{submission_id}", json.dumps({
                "status": "SE", "message": "System Error: Infrastructure temporarily unavailable."
            }))
        except Exception as r_err:
            logger.error(f"[!] Failed to publish SE to Redis: {r_err}")
        
    except Exception as e:
        logger.error(f"\n[!] CRITICAL PYTHON CRASH in queue_listener.process_submission:")
        logger.error(traceback.format_exc()) 
        
        if conn and cursor:
            try:
                conn.rollback()
                cursor.execute("UPDATE submissions SET status = 'SE', error_logs = 'System Error: An unexpected issue occurred during grading.' WHERE submission_id = %s", (submission_id,))
                conn.commit()
            except Exception as db_err:
                logger.error(f"[!] Failed to log SE crash to DB: {db_err}")
        
        try:
            redis_client.publish(f"submission_updates:{submission_id}", json.dumps({
                "status": "SE", "message": "System Error: An unexpected issue occurred during grading."
            }))
        except Exception as r_err:
            logger.error(f"[!] Failed to publish crash SE to Redis: {r_err}")
            
    finally:
        if cursor:
            cursor.close()
        if conn:
            release_db_connection(conn)

def route_job(submission_data):
    """The entry point for background threads. Routes jobs and handles thread-level crashes."""
    try:
        if submission_data.get('job_type') == 'moss_audit':
            contest_id = submission_data.get('contest_id')
            problem_id = submission_data.get('problem_id')
            logger.info(f"[+] Picked up MOSS Audit Job for Problem: {problem_id}")
            run_moss_audit(contest_id)
            
        elif submission_data.get('is_custom'):
            run_id = submission_data.get('run_id')
            logger.info(f"[+] Processing Custom Run: {run_id}")

            redis_client.publish(f"run_updates:{run_id}", json.dumps({"status": "Running"}))
            
            custom_tc = [{
                "test_case_id": "custom",
                "input_data": submission_data.get('custom_input', ''),
                "expected_output": ""
            }]
            
            lang = submission_data.get('language')
            limits = {'time': 1.0, 'memory': 1.0}

            result = grade_submission(
                submission_id=run_id,
                problem_id="custom",
                language=lang,
                source_code=submission_data.get('source_code'),
                source_s3_key=None, 
                test_cases=custom_tc,
                time_limit_ms=int(2000 * limits['time']),
                memory_limit_kb=int(256000 * limits['memory'])
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
                logger.info(f"[+] Picked up submission ID: {sub_id}")
                process_submission(sub_id)
                
    except Exception as e:
        logger.error(f"[!] CRITICAL THREAD CRASH handling job:")
        logger.error(traceback.format_exc())
        
        try:
            if submission_data and submission_data.get('is_custom'):
                run_id = submission_data.get('run_id')
                if run_id:
                    error_payload = json.dumps({
                        "status": "Completed", 
                        "output": "System error during custom run.", 
                        "verdict": "SE"
                    })
                    redis_client.set(f"run_result:{run_id}", error_payload, ex=600)
                    redis_client.publish(f"run_updates:{run_id}", error_payload)
        except Exception as recovery_err:
            logger.error(f"[!] Failed to push terminal error state for custom run: {recovery_err}")
            
    finally:
        job_semaphore.release()

def start_worker():
    """Main daemon loop that initializes infrastructure and pulls from Redis."""
    try:
        logger.info("[*] Performing startup infrastructure checks...")
        
        if not redis_client.ping():
            raise ConfigurationError("Redis ping failed.")
            
        test_conn = get_db_connection()
        release_db_connection(test_conn)
        
        get_s3_client() 
        
    except ConfigurationError as e:
        sys.stderr.write(f"FATAL STARTUP ERROR: {e}\n")
        sys.exit(1)
    except Exception as e:
        sys.stderr.write(f"FATAL STARTUP ERROR: Infrastructure connection failed: {e}\n")
        sys.exit(1)
    
    logger.info(f"[*] Worker started. Listening to Redis queue: '{QUEUE_NAME}'...")
    
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        while True:
            job_semaphore.acquire() 
            
            try:
                queue_result = redis_client.brpop(QUEUE_NAME, timeout=5)
                if queue_result:
                    _, message = queue_result
                    submission_data = json.loads(message)
                    executor.submit(route_job, submission_data)
                else:
                    job_semaphore.release()
                    
            except (redis.exceptions.ConnectionError, redis.exceptions.TimeoutError) as net_err:
                logger.error(f"[!] Redis network error: {net_err}. Retrying in 5 seconds...")
                time.sleep(5) 
                job_semaphore.release()
                
            except redis.exceptions.RedisError as cmd_err:
                logger.error(f"[!] Redis command/data error: {cmd_err}. Rate-limiting logs. Retrying in 2 seconds...")
                time.sleep(2) 
                job_semaphore.release()
                
            except json.JSONDecodeError as je:
                logger.error(f"[!] Dropping malformed JSON payload from queue: {je}")
                job_semaphore.release()
                
            except Exception as e:
                logger.error(f"[!] Unexpected error in main worker loop: {e}")
                logger.error(traceback.format_exc())
                time.sleep(1)
                job_semaphore.release()

if __name__ == "__main__":
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    start_worker()