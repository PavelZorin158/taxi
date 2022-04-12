FROM golang:1.17.8

LABEL maintainer="pasharick@gmail.com"

WORKDIR /usr/src/app/backend

RUN go get github.com/mattn/go-sqlite3

COPY . /usr/src/app

ENTRYPOINT go run main.go

EXPOSE 5005

VOLUME /usr/src/app/dir_db