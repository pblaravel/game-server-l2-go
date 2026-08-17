FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/loginserver ./cmd/loginserver \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gameserver ./cmd/gameserver \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/smoketest ./cmd/smoketest

FROM debian:bookworm-slim AS loginserver
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/loginserver /app/loginserver
COPY conf /app/conf
COPY sql /app/sql
EXPOSE 2107 9015
CMD ["/app/loginserver"]

FROM debian:bookworm-slim AS gameserver
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/gameserver /app/gameserver
COPY conf /app/conf
COPY sql /app/sql
COPY data /app/data
EXPOSE 7778
CMD ["/app/gameserver"]

FROM debian:bookworm-slim AS smoketest
WORKDIR /app
COPY --from=build /out/smoketest /app/smoketest
ENTRYPOINT ["/app/smoketest"]
