FROM golang:alpine as build-env

ENV GO111MODULE=on

RUN mkdir /docker_example
RUN mkdir -p /docker_example/proto

WORKDIR /docker_example

COPY ./proto/proto.pb.go /docker_example/proto
COPY ./main.go /docker_example

COPY go.mod .
COPY go.sum .
COPY . .

RUN go mod download

RUN go build -o docker_example .

CMD ./docker_example