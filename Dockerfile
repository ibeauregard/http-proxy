# syntax=docker/dockerfile:1
FROM golang:1.19.0-alpine3.16 as build-image
ARG APP_NAME
COPY go.mod /app/
WORKDIR /app
COPY . .
RUN go build -o $APP_NAME ./internal

FROM alpine:3.16
ARG APP_NAME
ARG CACHE_DIR_NAME
WORKDIR /home/$APP_NAME
COPY --from=build-image /app/$APP_NAME ./
RUN mkdir $CACHE_DIR_NAME
EXPOSE 8080
ENV APP_NAME ${APP_NAME}
CMD ["sh", "-c", "./$APP_NAME"]