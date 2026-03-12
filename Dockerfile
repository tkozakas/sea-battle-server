FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/server ./cmd/server/main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates wget

COPY --from=builder /bin/server /bin/server

ARG PORT=8080
ENV PORT=${PORT}

EXPOSE ${PORT}

HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD wget -q --spider http://localhost:${PORT}/health || exit 1

ENTRYPOINT ["/bin/server"]
