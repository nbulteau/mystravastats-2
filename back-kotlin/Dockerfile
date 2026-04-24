# Use the same Java major version declared in build.gradle.kts.
FROM gradle:9.4.1-jdk25 AS build

# Set the working directory
WORKDIR /app

# Copy the Gradle wrapper and the build files
COPY gradle/ gradle/
COPY build.gradle.kts settings.gradle.kts gradle.properties ./
# Copy the libraries
COPY libs/ libs/

# Copy the source code
COPY src/ src/


# Build the application
RUN gradle build --no-daemon -x test

# Use the same Java major version at runtime.
FROM eclipse-temurin:25-jre AS runtime

# Install curl for healthchecks and xdg-utils for optional browser handoff.
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl xdg-utils \
    && rm -rf /var/lib/apt/lists/*

ENV SERVER_ADDRESS=0.0.0.0
ENV SERVER_PORT=8080
ENV OPEN_BROWSER=false

# Set the working directory
WORKDIR /app

# Copy the built application from the build stage
COPY --from=build /app/build/libs/*.jar /app/mystravastats.jar
COPY famous-climb/ /app/famous-climb/

# Expose the port the application runs on
EXPOSE 8080
EXPOSE 8090

# Run the application
ENTRYPOINT ["java", "-jar", "/app/mystravastats.jar"]
