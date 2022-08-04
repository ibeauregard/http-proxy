# syntax=docker/dockerfile:1
FROM golang:1.19.0-alpine3.16 as build-image
COPY go.mod /app/
WORKDIR /app
COPY . .
RUN go build -o my_proxy ./internal

FROM alpine:3.16
WORKDIR /home/my_proxy
COPY --from=build-image /app/my_proxy ./
CMD ["./my_proxy"]