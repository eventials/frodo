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

type Broker interface {
    Close()
    Receive() (chan BrokerMessage, error)
}

type broker struct {
    connection *amqp.Connection
    channel *amqp.Channel
    queue *amqp.Queue
}

func NewBroker(url, queue string) (Broker, error) {
    conn, err := amqp.Dial(url)

    if err != nil {
        return nil, err
    }

    ch, err := conn.Channel()

    if err != nil {
        conn.Close()
        return nil, err
    }

    q, err := ch.QueueDeclare(queue, true, false, false, false, nil)

    if err != nil {
        ch.Close()
        return nil, err
    }

    b := &broker{conn, ch, &q}
    return b, nil
}

func (b *broker) Close() {
    b.connection.Close()
    b.channel.Close()
}

func (b *broker) Receive() (chan BrokerMessage, error) {
    msgs, err := b.channel.Consume(b.queue.Name, "", true, false, false, false, nil)

    if err != nil {
        return nil, err
    }

    ms := make(chan BrokerMessage)

    go func() {
        for m := range msgs {
            log.Println("Got new message from broker.")

            if m.ContentType == "application/json" {
                var brokerMessage BrokerMessage

                if err = json.Unmarshal(m.Body, &brokerMessage); err == nil {
                    ms <- brokerMessage
                } else {
                    log.Printf("Can't decode JSON message: %s", err)
                }
            } else {
                log.Printf("Message is not JSON: %s", m.ContentType)
            }
        }
    }()

    return ms, nil
}
