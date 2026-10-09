FROM node:22-bookworm-slim AS ui
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci --no-fund
COPY frontend/ ./
RUN npm run build

FROM golang:1.23-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=ui /src/internal/server/web/dist ./internal/server/web/dist
RUN CGO_ENABLED=0 go build -trimpath -o /policylens ./cmd/server
RUN ./scripts/install-kyverno.sh

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /policylens ./policylens
COPY --from=build /src/bin/kyverno ./bin/kyverno
ENV ADDR=0.0.0.0:8080
ENV KYVERNO_BIN=/app/bin/kyverno
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/app/policylens"]
