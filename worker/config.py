import os
import boto3
import redis
from psycopg2 import pool
from psycopg2.extras import RealDictCursor

class ConfigurationError(Exception):
    """Custom exception for missing environment variables to avoid raw sys.exit() calls."""
    pass

def get_required_env(var_name: str) -> str:
    value = os.getenv(var_name)
    if not value:
        raise ConfigurationError(f"Required environment variable '{var_name}' is missing or empty.")
    return value

# --- REDIS CONFIGURATION ---
# Safe at module level: redis-py connects lazily and won't crash on import if host is missing
redis_client = redis.Redis(
    host=os.getenv("REDIS_HOST", "redis"), 
    port=6379, 
    db=0, 
    decode_responses=True
)

# --- LAZY DATABASE CONFIGURATION ---
_db_pool = None

def get_db_connection():
    global _db_pool
    if _db_pool is None:
        # ISSUES 1 & 4 FIXED: Env vars are checked and the pool is created ONLY 
        # when a database connection is explicitly requested by the caller.
        db_config = {
            "dbname": "CampusCompile_db",
            "user": get_required_env("POSTGRES_USER"),
            "password": get_required_env("POSTGRES_PASSWORD"),
            "host": os.getenv("DB_HOST", "localhost"),
            "port": "5432"
        }
        _db_pool = pool.ThreadedConnectionPool(1, 20, **db_config, cursor_factory=RealDictCursor)
        print("[*] Database thread-pool created successfully.")
        
    return _db_pool.getconn()

def release_db_connection(conn):
    if _db_pool and conn:
        _db_pool.putconn(conn)

# --- LAZY S3 / MINIO CONFIGURATION ---
_s3_client = None
BUCKET_NAME = os.getenv("S3_BUCKET_NAME", "campus-testcases")

def get_s3_client():
    global _s3_client
    if _s3_client is None:
        # ISSUE 2 FIXED: Removed insecure default credentials. 
        # Forces explicit configuration, but safely deferred until S3 is actually needed.
        _s3_client = boto3.client(
            's3',
            endpoint_url=os.getenv('S3_ENDPOINT', 'http://minio:9000'),
            aws_access_key_id=get_required_env('S3_ACCESS_KEY'),
            aws_secret_access_key=get_required_env('S3_SECRET_KEY'),
            region_name='us-east-1' 
        )
    return _s3_client

def fetch_from_s3(s3_key: str, destination_path: str):
    try:
        client = get_s3_client()
        client.download_file(BUCKET_NAME, s3_key, destination_path)
    except Exception as e:
        # ISSUE 3 FIXED: Exception chaining ('from e') preserves the original 
        # boto3 traceback so you can accurately debug upstream failures.
        raise RuntimeError(f"Failed to fetch {s3_key} from S3") from e