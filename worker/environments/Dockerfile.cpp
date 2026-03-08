# Start from bare-bones Alpine
FROM alpine:latest

# Install g++ and standard C++ libraries without caching the download package (saves space)
RUN apk add --no-cache g++

WORKDIR /sandbox

# CRITICAL SECURITY: Explicitly map the student user to UID 1001 to match runner.py
RUN addgroup -g 1001 student_group && \
    adduser -D -u 1001 -G student_group student

# Drop root privileges
USER student

CMD ["/bin/sh"]