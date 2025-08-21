# Use an official Gradle image to build the application
FROM gradle:9.0.0-jdk23 AS build

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

# Use an official OpenJDK runtime as a parent image
FROM openjdk:23-jdk-slim AS runtime

# Install xdg-utils
RUN apt-get update && apt-get install -y xdg-utils

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