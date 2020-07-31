FROM golang:1.13-buster  AS build-env
ARG NAME

RUN  apt-get update && apt-get install git --yes && \
     mkdir -p /go/src/${NAME}/vendor && \
     mkdir -p /go/src/bin

ENV GO111MODULE=on
WORKDIR /go/src/${NAME}

# Manage Deps
COPY go.mod go.sum ./
RUN  go mod download

# Build src
COPY . .
RUN   GOOS=linux GOARCH=amd64  go build -o /go/src/bin/${NAME} ./

FROM debian:buster-slim
RUN apt-get update && apt-get install -y ca-certificates
ARG NAME
WORKDIR /app
COPY --from=build-env /go/src/bin/${NAME} /app
ENTRYPOINT ["./detector", "run"]
