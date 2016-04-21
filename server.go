package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "strconv"

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

func run(allowCors bool, cacheTTL int, cacheUrl, brokerUrl, brokerQueue, bindAddress string) {
    cache, err := storage.NewStorage(cacheUrl, cacheTTL)

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

    es := sse.NewEventSource(sse.Settings{
        AllowCors: allowCors,
        OnClientConnect: func (es sse.EventSource, c sse.Client) {
            // When a client connect for the first time, we send the last message.
            if msg, ok := es.GetLastMessage(c.Channel()); ok {
                c.SendMessage(msg)
            }
        },
        OnChannelCreate: func (es sse.EventSource, channelName string) {
            // Load the last message from cache to memory.
            if msg, err := cache.Get(channelName); err == nil {
                es.SetLastMessage(channelName, msg)
            }
        },
    })

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

    router.HandleFunc("/appstatus", func (w http.ResponseWriter, r *http.Request) {
        cacheOK := cache.Ping()
        brokerOK := b.Ping()
        statusOK := cacheOK && brokerOK
        w.Write([]byte(fmt.Sprintf("status:%t,cache:%t,broker:%t", statusOK, cacheOK, brokerOK)))
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
    var ttl int64
    var err error

    if ttl, err = strconv.ParseInt(defaultValue(os.Getenv("FRODO_TTL"), "60"), 10, 64); err != nil {
        ttl = 60
    }

    allowCors := flag.Bool("cors", os.Getenv("FRODO_CORS") == "true", "Allow CORS.")
    cacheUrl := flag.String("cache", defaultValue(os.Getenv("FRODO_CACHE"), "redis://127.0.0.1:6379/0"), "Cache URL.")
    cacheTTL := flag.Int("ttl", int(ttl), "Cache TTL in seconds.")
    brokerUrl := flag.String("broker", defaultValue(os.Getenv("FRODO_BROKER"), "amqp://"), "Broker URL.")
    brokerQueue := flag.String("queue", defaultValue(os.Getenv("FRODO_QUEUE"), "frodo"), "Broker Queue.")
    bindAddress := flag.String("bind", defaultValue(os.Getenv("FRODO_BIND"), ":3000"), "Bind Address.")

    flag.Parse()

    run(*allowCors, *cacheTTL, *cacheUrl, *brokerUrl, *brokerQueue, *bindAddress)
}
