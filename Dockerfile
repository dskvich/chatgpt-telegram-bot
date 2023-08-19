FROM golang:1.21-alpine as builder
RUN apk update && apk add --no-cache git

WORKDIR /app

COPY . ./
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go mod download
RUN go build -ldflags="-s -w" -o main main.go

# Create appuser.
ENV USER=appuser
ENV UID=10001
# See https://stackoverflow.com/a/55757473/12429735RUN
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

FROM scratch

WORKDIR /app

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /app/main ./

# Use an unprivileged user.
USER appuser:appuser

CMD ["./main"]