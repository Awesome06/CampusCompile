# Use a lightweight Alpine build for Java 21
FROM eclipse-temurin:21-jdk-alpine

WORKDIR /sandbox

# CRITICAL SECURITY: Explicitly map the student user to UID 1001 to match runner.py
RUN addgroup -g 1001 student_group && \
    adduser -D -u 1001 -G student_group student

# Drop root privileges
USER student

CMD ["/bin/sh"]