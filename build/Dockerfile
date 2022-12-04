FROM golang:1.19-alpine as go-builder

WORKDIR /app

COPY go.mod go.mod 

COPY go.sum go.sum 

RUN go mod download

COPY main.go main.go

COPY internal/ internal/

COPY generic/ generic/

RUN go build

FROM alpine:latest

COPY --from=go-builder app/ping-app /ping-app

ENTRYPOINT [ "./ping-app" ]