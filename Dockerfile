FROM golang:1.21-alpine as builder

# Add edge/testing repository for FFmpeg
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories

RUN apk update && apk add --no-cache git ffmpeg

WORKDIR /app

COPY . ./
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go mod download
RUN go build -ldflags="-s -w" -o main main.go


FROM alpine

WORKDIR /app

COPY --from=builder /app/main ./

CMD ["./main"]