# ==========================================
# Stage 1
# ==========================================
FROM golang:1.26.3-alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/hirely-api cmd/api/main.go

# ==========================================
# Stage 2
# ==========================================
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

ENV TZ=UTC
ENV PORT=8080

WORKDIR /root/

COPY --from=builder /app/bin/hirely-api .

EXPOSE $PORT

CMD ["./hirely-api"]
