FROM golang:1.13.4 AS builder
COPY . /source/
WORKDIR /source
RUN env GOOS=linux GOARCH=386 go build -o /tooling
CMD ["/tooling"]