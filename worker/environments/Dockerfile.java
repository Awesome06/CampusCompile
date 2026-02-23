# Use a lightweight Alpine build for Java 21
FROM eclipse-temurin:21-jdk-alpine

WORKDIR /sandbox

# Drop root privileges
RUN adduser -D student
USER student

CMD ["/bin/sh"]