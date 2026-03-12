FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/server ./cmd/server/main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates wget

COPY --from=builder /bin/server /bin/server

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD wget -q --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/bin/server"]
