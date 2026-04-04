# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /miniblue ./cmd/miniblue

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates
COPY --from=builder /miniblue /miniblue

EXPOSE 4566
ENV PORT=4566

ENTRYPOINT ["/miniblue"]
