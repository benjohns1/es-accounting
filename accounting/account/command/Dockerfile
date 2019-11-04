FROM es-accounting_go-builder:latest AS builder
ARG SERVICE_PATH=accounting/account
ARG SERVICE_NAME=command
COPY . /source/
WORKDIR /source/${SERVICE_PATH}/${SERVICE_NAME}
RUN env GOOS=linux GOARCH=386 go build -o /executable -tags dockernetwork

FROM scratch
COPY --from=builder /executable /
ENTRYPOINT ["/executable"]