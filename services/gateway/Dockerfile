FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY Backend/go.mod Backend/go.sum ./
RUN go mod download
COPY Backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gateway ./services/gateway/cmd/
# Note: This Dockerfile expects the services/ dir to be copied into Backend/ at build time
# In practice, we build from the monorepo root with the correct context

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /gateway .
EXPOSE 8080
ENTRYPOINT ["/app/gateway"]
