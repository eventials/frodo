// Package broker implements a message broker based on AMQP 0.9.1.
package broker

import (
    "encoding/json"
    "log"

    "github.com/streadway/amqp"
)

type BrokerMessage struct {
    Channel string `json:"channel"`
    Data json.RawMessage `json:"data"`
}

type Settings struct {
    Url string
    ExchangeName string
}

type Broker struct {
    settings Settings
    connection *amqp.Connection
    channel *amqp.Channel
    queue *amqp.Queue

    Message chan BrokerMessage
    ConnectionLost chan bool
}

func NewBroker(settings Settings) (*Broker, error) {
    conn, err := amqp.Dial(settings.Url)

    if err != nil {
        return nil, err
    }

    ch, err := conn.Channel()

    if err != nil {
        conn.Close()
        return nil, err
    }

    err = ch.ExchangeDeclare(settings.ExchangeName, "fanout", true, false, false, false, nil)

    if err != nil {
        ch.Close()
        return nil, err
    }

    q, err := ch.QueueDeclare("", false, false, true, false, nil)

    if err != nil {
        ch.Close()
        return nil, err
    }

    err = ch.QueueBind(q.Name, "", settings.ExchangeName, false, nil)

    if err != nil {
        ch.Close()
        return nil, err
    }

    b := &Broker{
        settings,
        conn,
        ch,
        &q,
        make(chan BrokerMessage),
        make(chan bool),
    }

    return b, nil
}

func (b *Broker) Close() {
    b.connection.Close()
    b.channel.Close()
    close(b.Message)
    close(b.ConnectionLost)
}

func (b *Broker) StartListen() error {
    msgs, err := b.channel.Consume(b.queue.Name, "", true, false, false, false, nil)

    if err != nil {
        return err
    }

    go func() {
        for m := range msgs {
            log.Println("Got new message from broker.")

            if m.ContentType == "application/json" {
                var brokerMessage BrokerMessage

                if err = json.Unmarshal(m.Body, &brokerMessage); err == nil {
                    log.Println("Valid message. Broadcasting...")
                    b.Message <- brokerMessage
                } else {
                    log.Printf("Can't decode JSON message: %s", err)
                }
            } else {
                log.Printf("Message is not JSON: %s", m.ContentType)
            }
        }
    }()

    return nil
}

func (b *Broker) Ping() bool {
    // TODO: Is this the best way to test?
    ch, err := b.connection.Channel()

    if err == nil {
        ch.Close()
        return true
    }

    b.ConnectionLost <- true

    log.Printf("Ping error: %s\n", err)

    go func() {
        // TODO: Should we panic if got an error
        // or should we try until some timeout?
        b.reconnect()
    }()

    return false
}

func (b *Broker) reconnect() error {
    log.Println("Reconnecting broker due connection loss.")

    newb, err := NewBroker(b.settings)

    if err != nil {
        return err
    }

    b.connection = newb.connection
    b.channel = newb.channel
    b.queue = newb.queue

    return b.StartListen()
}
