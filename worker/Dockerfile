# Use a slim Python image
FROM python:3.10-slim

# Prevent Python from writing .pyc files to disk
ENV PYTHONDONTWRITEBYTECODE=1
# Ensure print statements instantly appear in Docker logs
ENV PYTHONUNBUFFERED=1

# Install dependencies first for better layer caching
WORKDIR /app

# Install dependencies first for better layer caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the worker scripts
COPY queue_listener.py runner.py moss_auditor.py config.py ./

CMD ["python", "queue_listener.py"]