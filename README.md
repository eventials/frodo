# Frodo [![Build Status](https://travis-ci.org/eventials/frodo.svg?branch=master)](https://travis-ci.org/eventials/frodo) [![GoDoc](https://godoc.org/github.com/eventials/frodo?status.svg)](http://godoc.org/github.com/eventials/frodo)


Event Source Multiplexer

## About

Frodo is a dummy Event Source server. It only route messages from a broker to a channel over server-sent events.

Frodo uses AMQP protocol to receive messages and send it to a client. The message must be a valid JSON object, as below:

```js
{
    "channel": "/events/my-channel",
    "data": "some stuff"
}
```

The `channel` key must match a valid channel, or the message will be skipped.
The `data` key can be any valid JSON object.
This value is by-pass, so, Frodo won't take a look into it, it will just publish in the channel.
The client must know how to use this data.

Frodo uses a cache to store the last message receive in the channel.
This is used to broadcast the last message to new clients.

## Requirements

- RabbitMQ or any other AMQP 0.9.1 broker
- Redis

## Go dependencies

```sh
go get github.com/gorilla/mux
go get github.com/streadway/amqp
go get github.com/garyburd/redigo/redis
```

## Setup

For easy use, there's a Dockerfile to speed up things.
Just run `docker-composer up` and enjoy.

## Configuration

There's two ways to configure things, thought command line argumens or environment variables.

Command line arguments are priority if set, if not use environment variables, and if none available,
there's a default value for each option.

| Option       | CL Argument | Env Var  | Default                    |
|:-------------|:--------|:-------------|:---------------------------|
| Allow CORS   | -cors   | FRODO_CORS   | `false`                    |
| Bind Address | -bind   | FRODO_BIND   | `:3000`                    |
| Broker URL   | -broker | FRODO_BROKER | `amqp://`                  |
| Broker Queue | -queue  | FRODO_QUEUE  | `frodo`                    |
| Cache URL    | -cache  | FRODO_CACHE  | `redis://127.0.0.1:6379/0` |
| Cache TTL    | -ttl    | FRODO_TTL    | `60` seconds                |

## Building Docker Image

Just run `docker build -t username/imagename -f Dockerfile.build --no-cache .`.

To publish run `docker push username/imagename`.
