package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/gorilla/mux"

    "github.com/eventials/frodo/broker"
    "github.com/eventials/frodo/sse"
)

type ChannelStats struct {
    ClientCount int `json:"client_count"`
}

type Stats struct {
    ChannelCount int `json:"channel_count"`
    ClientCount int `json:"client_count"`
    Channels map[string]ChannelStats `json:"channels"`
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Frodo")
}

func run(appName, bindAddress, brokerUrl string) {
    es := sse.NewEventSource()
    log.Println("Event Source started.")
    defer es.Shutdown()

    b, err := broker.NewBroker(broker.Settings{brokerUrl, appName})

    if err != nil {
        log.Fatalf("Can't connect to broker: %s\n", err)
    }

    log.Println("Connected to broker.")
    defer b.Close()

    go func() {
        for _ = range b.ConnectionLost {
            log.Println("Broker connection lost. Closing channels...")
            es.CloseChannels()
        }
    }()

    go func() {
        for eventMessage := range b.Message {
            c := eventMessage.Channel
            msg := string(eventMessage.Data[:])
            es.SendMessage(c, msg)
        }
    }()

    err = b.StartListen()

    if err != nil {
        log.Fatalf("Can't receive messages from broker: %s\n", err)
    }

    router := mux.NewRouter()

    router.HandleFunc("/appstatus", func (w http.ResponseWriter, r *http.Request) {
        statusOK := b.Ping()

        if !statusOK {
            w.WriteHeader(http.StatusInternalServerError)
        }

        w.Write([]byte(fmt.Sprintf("status:%t", statusOK)))
    })

    router.HandleFunc("/api/stats", func (w http.ResponseWriter, r *http.Request) {
        channels := es.Channels()
        stats := Stats{
            len(channels),
            es.ConnectionCount(),
            make(map[string]ChannelStats),
        }

        for _, name := range channels {
            stats.Channels[name] = ChannelStats{
                es.ConnectionCountPerChannel(name),
            }
        }

        dump, err := json.Marshal(&stats)

        if err == nil {
            w.Header().Set("Content-Type", "application/json")
            w.Write(dump)
        } else {
            w.WriteHeader(http.StatusInternalServerError)
        }
    })

    router.Handle("/{channel:[a-z0-9-_/]+}", es)
    router.HandleFunc("/", indexHandler)

    log.Printf("Server started at %s.\n", bindAddress)
    http.ListenAndServe(bindAddress, router)
}

func defaultValue(a, b string) string {
    if len(a) == 0 {
        return b
    }

    return a
}

func main() {
    appName := flag.String("appname", defaultValue(os.Getenv("FRODO_NAME"), "frodo"), "Application name.")
    bindAddress := flag.String("bind", defaultValue(os.Getenv("FRODO_BIND"), ":3000"), "Bind Address.")
    brokerUrl := flag.String("broker", defaultValue(os.Getenv("FRODO_BROKER_URL"), "amqp://"), "Broker URL.")

    flag.Parse()

    run(*appName, *bindAddress, *brokerUrl)
}
