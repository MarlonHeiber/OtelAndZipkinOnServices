FROM golang:1.24  AS builder
WORKDIR /app
COPY . .
RUN GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -ldflags="-w -s" -o serverA .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/ .
EXPOSE 8080
CMD ["./serverA"]