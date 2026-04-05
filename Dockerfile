FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/moabukar/miniblue/internal/server.Version=${VERSION}" -o /miniblue ./cmd/miniblue
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /azlocal ./cmd/azlocal
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /healthcheck ./cmd/healthcheck

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /miniblue /miniblue
COPY --from=builder /azlocal /azlocal
COPY --from=builder /healthcheck /healthcheck
EXPOSE 4566 4567
ENV PORT=4566
USER 65534
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["/healthcheck"]
ENTRYPOINT ["/miniblue"]
