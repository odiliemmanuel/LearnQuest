# syntax=docker/dockerfile:1
# Build: docker build -t learnquest-backend ./backend
# Run:   docker run -p 8080:8080 --env-file backend/.env learnquest-backend

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Static build so it can run on distroless (no libc dependency).
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=build /out/server /app/server
# Seeded curriculum + public-domain book texts are read from disk relative to
# the working directory (internal/seed/content), so they must ship in the image.
COPY --from=build /src/internal/seed/content /app/internal/seed/content
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/server"]