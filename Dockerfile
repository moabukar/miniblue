FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /miniblue ./cmd/miniblue
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /healthcheck ./cmd/healthcheck

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /miniblue /miniblue
COPY --from=builder /healthcheck /healthcheck
EXPOSE 4566 4567
ENV PORT=4566
USER 65534
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["/healthcheck"]
ENTRYPOINT ["/miniblue"]
