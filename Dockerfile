# build backend
FROM golang:alpine AS build-env

WORKDIR /src
COPY go.* ./
RUN go mod download

COPY . ./
RUN mkdir -p ./dist/
RUN go build -o ./dist/main ./main.go


FROM alpine

ENV TZ=Europe/Moscow
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
RUN apk add --no-cache tzdata

WORKDIR /app

COPY --from=build-env /src/dist/main ./main
