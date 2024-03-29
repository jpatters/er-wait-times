FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o ermon .


FROM alpine:latest
LABEL org.opencontainers.image.source https://github.com/jpatters/er-wait-times
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/ermon .
RUN echo "0 * * * * /root/ermon > /proc/1/fd/1 2> /proc/1/fd/2" > /etc/crontabs/root
EXPOSE 8081
CMD crond -f
