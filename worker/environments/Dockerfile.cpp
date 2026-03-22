FROM alpine:latest

# Install g++ and standard C++ libraries
RUN apk add --no-cache g++

# Create the student user FIRST
RUN addgroup -g 1001 student_group && \
    adduser -D -u 1001 -G student_group student

# Create the shared directory and give ownership to the student
RUN mkdir -p /sandbox_shared && chown student:student_group /sandbox_shared

WORKDIR /sandbox_shared

# Drop root privileges
USER student

CMD ["/bin/sh"]