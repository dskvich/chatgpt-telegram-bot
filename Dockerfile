FROM golang:1.20-alpine as builder
RUN apk update && apk add --no-cache git

WORKDIR /app

COPY . ./
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o main main.go

FROM alpine

WORKDIR /app

COPY --from=builder /app/main ./

# Run under non-privileged user with minimal write permissions
RUN adduser -S -D -H user
USER user

CMD ["./main"]