FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o ermon .


FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/ermon .
RUN echo "0 * * * * /root/ermon > /proc/1/fd/1 2> /proc/1/fd/2" > /etc/crontabs/root
EXPOSE 8081
CMD crond -f
