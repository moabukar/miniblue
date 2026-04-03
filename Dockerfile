# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /local-azure ./cmd/local-azure

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates
COPY --from=builder /local-azure /local-azure

EXPOSE 4566
ENV PORT=4566

ENTRYPOINT ["/local-azure"]
