FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/moabukar/miniblue/internal/server.Version=${VERSION}" -o /miniblue ./cmd/miniblue
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /azlocal ./cmd/azlocal
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /healthcheck ./cmd/healthcheck
RUN mkdir -p /home/nonroot

FROM scratch AS slim
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /miniblue /miniblue
COPY --from=builder /azlocal /azlocal
COPY --from=builder /healthcheck /healthcheck
COPY --from=builder --chown=65534:65534 /home/nonroot /home/nonroot
EXPOSE 4566 4567
ENV PORT=4566
ENV HOME=/home/nonroot
USER 65534
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["/healthcheck"]
ENTRYPOINT ["/miniblue"]

# `full` adds the docker CLI so AKS_BACKEND=k3s and ACI's real backend can
# shell out when miniblue itself runs in a container with the host docker
# socket mounted. Build with: docker build --target=full -t miniblue:full .
FROM alpine:3.20 AS full
RUN apk add --no-cache docker-cli ca-certificates
COPY --from=builder /miniblue /miniblue
COPY --from=builder /azlocal /azlocal
COPY --from=builder /healthcheck /healthcheck
COPY --from=builder --chown=65534:65534 /home/nonroot /home/nonroot
EXPOSE 4566 4567
ENV PORT=4566
ENV HOME=/home/nonroot
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["/healthcheck"]
ENTRYPOINT ["/miniblue"]

FROM slim
