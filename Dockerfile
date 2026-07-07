# ---- Build stage ----
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy module files first
COPY go.mod go.sum ./

# Force Go modules on and download
ENV GO111MODULE=on
RUN go mod download

# Copy the rest of the source
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/main.go

# ---- Runtime stage ----
FROM alpine:3.19

WORKDIR /app

# ca-certificates is required for HTTPS calls to Treasury.gov
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]