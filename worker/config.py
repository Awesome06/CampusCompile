import os
import sys
import boto3
import redis
import psycopg2
from psycopg2.extras import RealDictCursor

# --- STRICT ENVIRONMENT VALIDATION ---
def get_required_env(var_name: str) -> str:
    value = os.getenv(var_name)
    if not value:
        sys.stderr.write(f"FATAL STARTUP ERROR: Required environment variable '{var_name}' is missing or empty.\n")
        sys.exit(1)
    return value

# --- DATABASE CONFIGURATION ---
DB_CONFIG = {
    "dbname": "CampusCompile_db",
    "user": get_required_env("POSTGRES_USER"),
    "password": get_required_env("POSTGRES_PASSWORD"),
    "host": os.getenv("DB_HOST", "localhost"),
    "port": "5432"
}

def get_db_connection():
    return psycopg2.connect(**DB_CONFIG, cursor_factory=RealDictCursor)

# --- REDIS CONFIGURATION ---
redis_client = redis.Redis(
    host=os.getenv("REDIS_HOST", "redis"), 
    port=6379, 
    db=0, 
    decode_responses=True
)

# --- S3 / MINIO CONFIGURATION ---
s3_client = boto3.client(
    's3',
    endpoint_url=os.getenv('S3_ENDPOINT', 'http://minio:9000'),
    aws_access_key_id=os.getenv('S3_ACCESS_KEY', 'campus_admin'),
    aws_secret_access_key=os.getenv('S3_SECRET_KEY', 'campus_password'),
    region_name='us-east-1' 
)
BUCKET_NAME = "campus-testcases"

def fetch_from_s3(s3_key: str, destination_path: str):
    try:
        s3_client.download_file(BUCKET_NAME, s3_key, destination_path)
    except Exception as e:
        # Raise instead of silently printing so the caller (like the grader) knows it failed
        raise Exception(f"Failed to fetch {s3_key} from S3: {e}")