package broker

import (
    "os"
    "testing"
    "time"

    "github.com/streadway/amqp"
)

func TestReceiveMessage(t *testing.T) {
    b, err := NewBroker(os.Getenv("AMQP_URL"), "queue")

    if err != nil {
        t.Fatal(err)
    }

    defer b.Close()

    msgs, err := b.Receive()

    if err != nil {
        t.Fatal(err)
    }

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

    for {
        select {
            case eventMessage := <- msgs:
                if eventMessage.Channel != "/test/channel" {
                    t.Fatal("Wrong channel")
                }

                msg := string(eventMessage.Data[:])

                if msg != "\"test message\"" {
                    t.Fatal("Wrong message")
                }

                return
            case <-time.After(5 * time.Second):
                t.Fatal("No message receive. Timeout.")
        }
    }
}