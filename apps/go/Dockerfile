# syntax=docker/dockerfile:1

# ---- builder ----
FROM golang:1.25 AS builder

WORKDIR /src

# Cache module downloads separately from source for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static build so it runs on distroless/static (no libc).
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- runtime ----
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /out/api /app/api
# Migrations are read at startup via file://migrations, resolved relative to WORKDIR.
COPY --from=builder /src/migrations /app/migrations

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/app/api"]
