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
    "github.com/eventials/frodo/storage"
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

func run(cacheUrl, brokerUrl, brokerQueue, bindAddress string) {
    cache, err := storage.NewStorage(cacheUrl)

    if err != nil {
        log.Fatalf("Can't connect to storage: %s", err)
    }

    log.Println("Connected to storage.")
    defer cache.Close()

    b, err := broker.NewBroker(brokerUrl, brokerQueue)

    if err != nil {
        log.Fatalf("Can't connect to broker: %s", err)
    }

    log.Println("Connected to broker.")
    defer b.Close()

    es := sse.NewEventSource(func (c sse.Client) {
        // When a client connect for the first time,
        // we send the last message.
        if msg, err := cache.Get(c.Channel()); err == nil {
            c.SendMessage(msg)
        }
    }, nil)

    defer es.Shutdown()

    msgs, err := b.Receive()

    if err != nil {
        log.Fatalf("Can't receive messages: %s\n", err)
    }

    // Listen for incoming messages from Broker.
    // If got any, store in the cache and send to ES listeners.
    go func () {
        for {
            eventMessage := <- msgs

            c := eventMessage.Channel
            msg := string(eventMessage.Data[:])
            cache.Set(c, msg)
            es.SendMessage(c, msg)
        }
    }()

    router := mux.NewRouter()
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

    log.Printf("Server started at %s.", bindAddress)
    http.ListenAndServe(bindAddress, router)
}

func defaultValue(a, b string) string {
    if len(a) == 0 {
        return b
    }

    return a
}

func main() {
    cacheUrl := flag.String("cache", defaultValue(os.Getenv("FRODO_CACHE"), ":6379"), "Cache URL.")
    brokerUrl := flag.String("broker", defaultValue(os.Getenv("FRODO_BROKER"), "amqp://"), "Broker URL.")
    brokerQueue := flag.String("queue", defaultValue(os.Getenv("FRODO_QUEUE"), "frodo"), "Broker Queue.")
    bindAddress := flag.String("bind", defaultValue(os.Getenv("FRODO_BIND"), ":3000"), "Bind Address.")

    flag.Parse()

    run(*cacheUrl, *brokerUrl, *brokerQueue, *bindAddress)
}