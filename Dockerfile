FROM golang:1.14-alpine  AS build-env
ARG NAME

RUN  apk update && apk add git && \
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

FROM alpine
RUN apk update && apk upgrade ca-certificates
ARG NAME
WORKDIR /app
COPY --from=build-env /go/src/bin/${NAME} /app
ENTRYPOINT ["/app/h2d"]
