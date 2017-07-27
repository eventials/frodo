# Frodo [![Build Status](https://travis-ci.org/eventials/frodo.svg?branch=master)](https://travis-ci.org/eventials/frodo) [![GoDoc](https://godoc.org/github.com/eventials/frodo?status.svg)](http://godoc.org/github.com/eventials/frodo)


Event Source Multiplexer

## About

Frodo is a dummy Event Source server.


The `channel` key must match a valid channel, or the message will be skipped.
The `data` key can be any valid JSON object.
This value is by-pass, so, Frodo won't take a look into it, it will just publish in the channel.
The client must know how to use this data.
