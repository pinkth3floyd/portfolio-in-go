# Multi-stage build for Go + HTMX portfolio

FROM node:20-alpine AS css
WORKDIR /app/tailwind
COPY tailwind/package.json ./
RUN npm install
COPY tailwind/ ./
COPY web/templates /app/web/templates
RUN npm run build

FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=css /app/web/static/css/app.css ./web/static/css/app.css
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/server /app/server
COPY web /app/web
COPY migrations /app/migrations
COPY fixtures /app/fixtures
RUN mkdir -p /app/data /app/web/static/uploads
ENV ADDR=:3000 \
    DATA_DIR=/app/data \
    DB_PATH=/app/data/app.db \
    TEMPLATES_DIR=/app/web/templates \
    STATIC_DIR=/app/web/static \
    MIGRATIONS_DIR=/app/migrations \
    FIXTURES_PATH=/app/fixtures/seed.json \
    SEED_ON_EMPTY=true
EXPOSE 3000
VOLUME ["/app/data"]
CMD ["/app/server"]
