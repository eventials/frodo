FROM golang:1.6

RUN apt-get update && apt-get install -y wget
RUN wget https://github.com/jwilder/dockerize/releases/download/v0.2.0/dockerize-linux-amd64-v0.2.0.tar.gz
RUN tar -C /usr/local/bin -xzvf dockerize-linux-amd64-v0.2.0.tar.gz

RUN mkdir -p /go/src/github.com/eventials/frodo
WORKDIR /go/src/github.com/eventials/frodo

RUN go get github.com/gorilla/mux
RUN go get github.com/streadway/amqp
RUN go get github.com/getsentry/raven-go
