# Use a slim Python image
FROM python:3.10-slim

WORKDIR /app

# Install the required Python packages for your worker
COPY requirements.txt .

RUN pip install --no-cache-dir -r requirements.txt

# Copy your worker scripts into the container
COPY queue_listener.py runner.py ./

# Start the worker with unbuffered output so we can see print statements in real-time
CMD ["python", "-u", "queue_listener.py"]