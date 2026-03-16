import os
import shutil
import mosspy
import requests
from bs4 import BeautifulSoup
from typing import List, Dict
import time
from config import redis_client, get_db_connection, release_db_connection

# 👇 NEW: Import everything directly from our centralized config
from config import get_db_connection, fetch_from_s3, redis_client

# --- CONFIGURATION ---
# You must register for a MOSS ID by emailing: moss@moss.stanford.edu
MOSS_USER_ID = os.getenv("MOSS_USER_ID", "YOUR_MOSS_ID_HERE") 

# Map CampusCompile language tags to MOSS language tags
LANGUAGE_MAP = {
    'cpp': 'cc',
    'java': 'java',
    'python': 'python'
}

def parse_moss_report(moss_url: str) -> List[Dict]:
    """Scrapes the MOSS HTML report to extract flagged user pairs and their similarity scores."""
    parsed_results = []
    try:
        response = requests.get(moss_url)
        soup = BeautifulSoup(response.text, 'html.parser')
        
        # MOSS results are strictly in an HTML table
        table = soup.find('table')
        if not table:
            return parsed_results

        rows = table.find_all('tr')[1:] # Skip header
        for row in rows:
            cols = row.find_all('td')
            if len(cols) >= 2:
                # MOSS formats links like: "user_id_1.cpp (85%)"
                link1 = cols[0].find('a').text.strip()
                link2 = cols[1].find('a').text.strip()
                
                # Extract User IDs
                user1 = link1.split('.')[0]
                user2 = link2.split('.')[0]
                
                # Extract Percentages
                score1 = int(link1.split('(')[1].replace('%)', ''))
                score2 = int(link2.split('(')[1].replace('%)', ''))
                
                # Use the highest similarity score between the pair
                max_score = max(score1, score2)
                
                # Only flag pairs with > 40% structural similarity to reduce noise
                if max_score > 40:
                    parsed_results.append({
                        "user_1": user1,
                        "user_2": user2,
                        "score": max_score
                    })
    except Exception as e:
        print(f"[!] Failed to parse MOSS HTML: {e}")
        
    return parsed_results

def run_moss_audit(contest_id: str):
    print(f"\n[*] Starting Automated MOSS Audit for Contest: {contest_id}")
    
    conn = get_db_connection()
    cursor = conn.cursor()
    
    try:
        # 1. Strict Environment Validation -> Triggers 'failed' state
        if MOSS_USER_ID == "YOUR_MOSS_ID_HERE" or not MOSS_USER_ID.strip():
            print("[!] FATAL: MOSS_USER_ID is missing or invalid. Audit Failed.")
            cursor.execute("UPDATE contests SET moss_audit_status = 'failed' WHERE contest_id = %s", (contest_id,))
            conn.commit()
            
            # 👇 CHANGED: Replaced local instantiation with centralized client
            redis_client.sadd("dirty_contests", contest_id)
            return

        cursor.execute("SELECT DISTINCT problem_id FROM submissions WHERE contest_id = %s", (contest_id,))
        problems = [row['problem_id'] for row in cursor.fetchall()]

        for prob_id in problems:
            cursor.execute("""
                SELECT DISTINCT ON (user_id) user_id, language, source_code_s3_key
                FROM submissions
                WHERE contest_id = %s AND problem_id = %s AND source_code_s3_key IS NOT NULL
                ORDER BY user_id, submitted_at DESC
            """, (contest_id, prob_id))
            
            submissions = cursor.fetchall()
            if not submissions:
                continue

            grouped_subs = {'cpp': [], 'java': [], 'python': []}
            for sub in submissions:
                lang = sub['language']
                if lang in grouped_subs:
                    grouped_subs[lang].append(sub)

            for lang, subs in grouped_subs.items():
                if len(subs) < 2:
                    continue 
                    
                print(f"[*] Analyzing {len(subs)} {lang.upper()} submissions...")
                
                batch_dir = f"/tmp/moss_{contest_id}_{prob_id}_{lang}"
                os.makedirs(batch_dir, exist_ok=True)
                
                moss_lang = LANGUAGE_MAP[lang]
                m = mosspy.Moss(MOSS_USER_ID, moss_lang)
                
                for sub in subs:
                    user_id = sub['user_id']
                    s3_key = sub['source_code_s3_key']
                    
                    file_ext = "cpp" if lang == "cpp" else "py" if lang == "python" else "java"
                    local_path = os.path.join(batch_dir, f"{user_id}.{file_ext}")
                    
                    fetch_from_s3(s3_key, local_path)
                    m.addFile(local_path)
                    
                print("[*] Uploading to Stanford MOSS servers (this may take a minute)...")
                url = m.send()
                print(f"[+] MOSS Report URL: {url}")
                
                flagged_pairs = parse_moss_report(url)
                
                for pair in flagged_pairs:
                    cursor.execute("""
                        INSERT INTO plagiarism_reports (contest_id, problem_id, user_1_id, user_2_id, similarity_score, moss_url)
                        VALUES (%s, %s, %s, %s, %s, %s)
                    """, (contest_id, prob_id, pair['user_1'], pair['user_2'], pair['score'], url))
                    
                conn.commit()
                print(f"[+] Logged {len(flagged_pairs)} flagged pairs to the database.")
                
                shutil.rmtree(batch_dir, ignore_errors=True)

            print("[*] Sleeping for 20 seconds to respect Stanford MOSS rate limits...")
            time.sleep(20)

        # 2. Finalize Database State to 'completed' on Success
        cursor.execute("UPDATE contests SET moss_audit_status = 'completed' WHERE contest_id = %s", (contest_id,))
        conn.commit()
        print(f"[+] Contest {contest_id} audit finalized. Status set to 'completed'.")

        # 👇 CHANGED: Replaced local instantiation with centralized client
        redis_client.sadd("dirty_contests", contest_id)

    except Exception as e:
        print(f"[!] MOSS Audit crashed: {e}")
        # Rollback any half-finished transactions
        conn.rollback() 
        
        # 3. Explicitly mark as failed in the event of a runtime crash
        try:
            cursor.execute("UPDATE contests SET moss_audit_status = 'failed' WHERE contest_id = %s", (contest_id,))
            conn.commit()
            
            # 👇 CHANGED: Replaced local instantiation with centralized client
            redis_client.sadd("dirty_contests", contest_id)
        except Exception as inner_e:
            print(f"[!] Failed to update crash status to database: {inner_e}")
            
    finally:
        cursor.close()
        release_db_connection(conn)