FROM golang:1.23-alpine as golang_build

ENV APP_DIR /app
WORKDIR /app

VOLUME $APP_DIR

RUN apk add build-base

COPY go.mod .
COPY go.sum .
RUN go mod download && go mod tidy

COPY . .