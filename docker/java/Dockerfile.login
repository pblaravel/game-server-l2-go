# Build the vendored Java login server (aCis / l2-unity).
FROM gradle:8.14.3-jdk21 AS build
WORKDIR /src
COPY . .
RUN gradle --no-daemon --no-parallel installDist

FROM eclipse-temurin:21-jre-jammy
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends netcat-openbsd \
 && rm -rf /var/lib/apt/lists/*
COPY --from=build /src/build/install/l2-unity-loginserver /opt/ls
COPY conf /app/conf
EXPOSE 2107 9015
CMD ["java", "-cp", "/opt/ls/lib/*", "com.shnok.javaserver.Main"]
