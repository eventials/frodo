package broker

import (
    "os"
    "testing"
    "time"

    "github.com/streadway/amqp"
)

func TestReceiveMessage(t *testing.T) {
    received := make(chan bool)

    b, err := NewBroker(Settings{
        Url: os.Getenv("AMQP_URL"),
        ExchangeName: "queue",
        OnMessage: func (msg BrokerMessage) {
            if msg.Channel != "/test/channel" {
                t.Fatal("Wrong channel")
            }

            text := string(msg.Data[:])

            if text != "\"test message\"" {
                t.Fatal("Wrong message")
            }

            received <- true
        },
    })

    if err != nil {
        t.Fatal(err)
    }

    defer b.Close()

    err = b.StartListen()

    if err != nil {
        t.Fatal(err)
    }

    go func() {
        for {
            select {
                case <- received:
                    return
                case <-time.After(5 * time.Second):
                    t.Fatal("No message receive. Timeout.")
            }
        }
    }()

    go func() {
        time.Sleep(1 * time.Second)

        conn, err := amqp.Dial(os.Getenv("AMQP_URL"))

        if err != nil {
            t.Fatal(err)
        }

        defer conn.Close()

        ch, err := conn.Channel()

        if err != nil {
            t.Fatal(err)
        }

        err = ch.Publish(
            "queue",
            "",
            false,
            false,
            amqp.Publishing {
                ContentType: "application/json",
                Body:        []byte("{\"channel\":\"/test/channel\",\"data\":\"test message\"}"),
            },
        )

        if err != nil {
            t.Fatal(err)
        }
    }()
}