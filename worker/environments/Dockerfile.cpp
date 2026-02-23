# Start from bare-bones Alpine
FROM alpine:latest

# Install g++ and standard C++ libraries without caching the download package (saves space)
RUN apk add --no-cache g++

WORKDIR /sandbox

# Drop root privileges
RUN adduser -D student
USER student

CMD ["/bin/sh"]